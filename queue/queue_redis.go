package queue

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

/*
 * @Time   : 2021/1/16 上午11:20
 * @Email  : jingjing.yang@tvb.com
 */

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// 基于redis实现队列机制：
// 一、原理
//    redis链表右边压入数据左边弹出数据实现`先进后出`队列，redis有序集合的分值字段记录延时执行时间到达执行时刻就执行任务实现延时队列
// 二、producer
// 	  实时队列：往redis链表（list） rpush 数据
//    延时队列：往redis有序集合（sorted set）zadd数据
// 三、consumer/worker步骤
//    step1、调度延迟任务，从延迟有序集合（queueName:delayed）取出Score值小于等于当前时间戳的延迟任务丢到List队列
//    step2、处理失败重试任务：从保留有序集合（queueName:reserved）取出Score值小于等于当前时间戳的保留任务丢到List队列
//    step3、调度list尝试执行：从list取出1条，将字段Attempts自增1，Score值为任务执行超时的时间戳，丢到保留有序集合
//    step4、判断丢到保留有序集合是否成功，以及任务执行次数是否超限，超限走失败流程，正常走执行流程
//    step5、执行超时or执行失败，立即走重试逻辑（从保留有序集合删除，丢到延迟有序集合）；执行成功任务结束（从保留有序集合删除）
// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

// redisQueue 基于Redis实现的队列
// implement QueueIFace
type redisQueue struct {
	queueBasic               // 队列基础可公用方法
	connection *redis.Client // connection redis客户端实例
	luaScripts *luaScripts   // redis lua脚本生成器
}

// Size 获取队列长度
func (r *redisQueue) Size(queue string) (size int64) {
	ctx := context.Background()
	result, _ := r.luaScripts.Size().Run(
		ctx,
		r.connection,
		[]string{r.name(queue), r.delayedName(queue), r.reservedName(queue)},
	).Int64()
	return result
}

// Push 投递一条任务到队列
func (r *redisQueue) Push(queue string, payload interface{}) (err error) {
	ctx := context.Background()
	return r.connection.RPush(ctx, queue, payload).Err()
}

// Later 延迟指定时长后执行的延迟任务
func (r *redisQueue) Later(queue string, durationTo time.Duration, payload interface{}) (err error) {
	return r.LaterAt(queue, time.Now().Add(durationTo), payload)
}

// LaterAt 指定时刻执行的延时任务
func (r *redisQueue) LaterAt(queue string, timeAt time.Time, payload interface{}) (err error) {
	item := redis.Z{
		Score:  float64(timeAt.Unix()),
		Member: payload,
	}
	ctx := context.Background()
	return r.connection.ZAdd(ctx, r.delayedName(queue), &item).Err()
}

// Pop 取出弹出一条待执行的任务
func (r *redisQueue) Pop(queue string) (job JobIFace, exist bool) {
	// step1、调度延迟任务，从延迟有序集合（queueName:delayed）取出Score值小于等于当前时间戳的延迟任务丢到List队列
	// step2、处理失败重试任务：从保留有序集合（queueName:reserved）取出Score值小于等于当前时间戳的保留任务丢到List队列
	// step3、调度list尝试执行：从list取出1条，将字段Attempts自增1，Score值为任务执行超时的时间戳，丢到保留有序集合（queueName:reserved）

	now := time.Now()

	// step1、migrate expired delay zSet data to queue list
	ctx := context.Background()
	r.luaScripts.MigrateExpiredJobs().Run(
		ctx,
		r.connection,
		[]string{r.delayedName(queue), r.name(queue)},
		now.Unix(),
	)

	// step2、migrate expired reserved zSet data to queue list
	r.luaScripts.MigrateExpiredJobs().Run(
		ctx,
		r.connection,
		[]string{r.reservedName(queue), r.name(queue)},
		now.Unix(),
	)

	// step3、get one item from queue list
	ret3, err := r.luaScripts.Pop().Run(
		ctx,
		r.connection,
		[]string{r.name(queue), r.reservedName(queue)}, // 从list移动到reserved的zSet
		now.Unix(), // 当前时间戳，用于填充为0的首次取出时间（PopTime字段）
	).Result()

	if err != nil {
		// redis pop lua execute error
		return nil, false
	}

	// set payload
	jobAndReserved := ret3.([]interface{})
	if len(jobAndReserved) != 2 {
		// array result returned
		return nil, false
	}
	if jobAndReserved[0] == nil || jobAndReserved[1] == nil {
		// job or reserved job is nil
		return nil, false
	}

	// transform type format
	var rJob, reserved Payload
	if r.unmarshalPayload([]byte(jobAndReserved[0].(string)), &rJob) != nil {
		return nil, false
	}
	if r.unmarshalPayload([]byte(jobAndReserved[1].(string)), &reserved) != nil {
		return nil, false
	}

	// set job timeoutAt
	// rJob.TimeoutAt = now.Add(time.Duration(reserved.Timeout) * time.Second).Unix()
	return &JobRedis{
		redis:      r.connection,
		lock:       sync.Mutex{},
		luaScripts: r.luaScripts,
		jobProperty: jobProperty{
			handler:    r,
			name:       queue,
			job:        jobAndReserved[0].(string),
			reserved:   jobAndReserved[1].(string),
			payload:    &rJob,
			isReleased: false,
			isDeleted:  false,
			hasFailed:  false,
			popTime:    time.Unix(reserved.PopTime, 0),
			timeout:    time.Duration(reserved.Timeout) * time.Second,
			timeoutAt:  now.Add(time.Duration(reserved.Timeout) * time.Second),
		},
	}, true
}

// SetConnection
// 设置redis队列的连接器：redis client句柄指针
func (r *redisQueue) SetConnection(connection interface{}) (err error) {
	r.connection = connection.(*redis.Client)
	return nil
}

// GetConnection
// 获取redis队列的连接器：redis client句柄指针（interface）使用前需显式转换
// example:
// 		conn, _ := r.GetConnection()
// 		client := conn.(*redis.Client)
//		client.Set("key", "values")
func (r *redisQueue) GetConnection() (connection interface{}, err error) {
	if r.connection == nil {
		return nil, errors.New("null pointer connection instance")
	}

	return r.connection, nil
}
