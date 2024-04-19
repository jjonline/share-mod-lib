package queue

import (
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
	"time"
)

/*
 * @Time   : 2021/02/09 11:00
 * @Email  : jingjing.yang@tvb.com
 */

type JobRedis struct {
	basic      queueBasic // 引入基础公用方法
	redis      *redis.Client
	luaScripts *luaScripts
	lock       sync.Mutex // 防幻读锁
	jobProperty
}

// Release 释放任务job：job重新再试--从reserved有序集合丢到delayed延迟有序集合
func (job *JobRedis) Release(delay int64) (err error) {
	job.lock.Lock()
	defer job.lock.Unlock()

	job.isReleased = true

	ctx := context.Background()
	// delete reserved zSet, then push it to delayed zSet
	err = job.luaScripts.Release().Run(
		ctx,
		job.redis,
		[]string{job.basic.delayedName(job.name), job.basic.reservedName(job.name)},
		job.reserved,
		time.Now().Add(time.Duration(delay)*time.Second).Unix(),
	).Err()

	return err
}

// Delete 删除任务job：任务不再执行--从reserved有序集合删除
func (job *JobRedis) Delete() (err error) {
	job.lock.Lock()
	defer job.lock.Unlock()
	job.isDeleted = true

	// delete reserved job from zSet
	ctx := context.Background()
	err = job.redis.ZRem(ctx, job.basic.reservedName(job.name), job.reserved).Err()

	return err
}

func (job *JobRedis) IsDeleted() (deleted bool) {
	job.lock.Lock()
	defer job.lock.Unlock()
	return job.isDeleted
}

func (job *JobRedis) IsReleased() (released bool) {
	job.lock.Lock()
	defer job.lock.Unlock()
	return job.isReleased
}

// Attempts 获取当前job已被尝试执行的次数
func (job *JobRedis) Attempts() (attempt int64) {
	return job.payload.Attempts + 1
}

// PopTime 任务job首次被执行的时刻
func (job *JobRedis) PopTime() (time time.Time) {
	return job.popTime
}

// Timeout 任务超时时长
func (job *JobRedis) Timeout() (time time.Duration) {
	return job.jobProperty.timeout
}

// TimeoutAt 任务job执行超时的时刻
func (job *JobRedis) TimeoutAt() (time time.Time) {
	return job.jobProperty.timeoutAt
}

func (job *JobRedis) HasFailed() (hasFail bool) {
	job.lock.Lock()
	defer job.lock.Unlock()
	return job.hasFailed
}

func (job *JobRedis) MarkAsFailed() {
	job.lock.Lock()
	defer job.lock.Unlock()
	job.hasFailed = true
}

func (job *JobRedis) Failed(err error) {
	// redis技术栈下实现的队列失败没有后续动作
	// 任务失败外部记录通过初始化队列时调用 SetFailedJobHandler 设置
	return
}

func (job *JobRedis) GetName() (queueName string) {
	return job.name
}

func (job *JobRedis) Queue() (queue QueueIFace) {
	return job.handler
}

func (job *JobRedis) Payload() (payload *Payload) {
	return job.payload
}
