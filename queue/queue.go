package queue

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// queue队列支持的底层驱动名称常量
// 后续扩充mq、sqs、db等在此添加常量并实现 QueueIFace 接口予以关联
const (
	Redis  = "redis"
	Memory = "memory"
)

// Queue 队列struct
type Queue struct {
	queueBasic            // 引入队列基础方法
	driver     string     // 记录底层队列实现
	queue      QueueIFace // 底层队列实现实体类，指针类型interface
	manager    *manager   // 管理者对象实例
	logger     Logger     // 队列日志记录器，统一固定使用zap
}

// New 初始化一个队列
// 	@param driver     队列实现底层驱动，可选值见上方14行附近位置的常量
// 	@param conn       driver对应底层驱动连接器句柄，具体类型参考 QueueIFace 实体类
// 	@param logger     实现 Logger 接口的结构体实例的指针对象
// 	@param concurrent 单个队列最大并发消费数
func New(driver string, conn interface{}, logger Logger, concurrent int64) *Queue {
	var queue QueueIFace

	// init specify queue driver
	switch driver {
	case Memory:
		queue = &memoryQueue{lock: sync.Mutex{}}
	case Redis:
		// queue = &redisQueue{connection: conn.(*redis.Client)}
		queue = &redisQueue{luaScripts: &luaScripts{}}
	default:
		panic("do not implement queue instance: " + driver)
	}

	// set connection
	err := queue.SetConnection(conn)
	if nil != err {
		panic(err.Error())
	}

	return &Queue{
		driver:  driver,
		queue:   queue,
		manager: newManager(queue, logger, concurrent),
		logger:  logger,
	}
}

// region 处理失败任务Failed相关方法

// SetFailedJobHandler 设置失败任务的收尾处理器
// 1、尝试了指定的最大尝试次数后仍然失败的任务善后方法
// 2、此时通过此处设置的处理器可记录到底哪个任务失败了以及失败任务的payload参数情况
// 3、以及后续的重试等逻辑等
func (q *Queue) SetFailedJobHandler(failedJobHandler FailedJobHandler) {
	q.manager.failedJobHandler = failedJobHandler
}

// endregion

// region 注册任务类相关方法

// BootstrapOne boot注册载入一个队列任务
//  @param task 任务类实例指针
func (q *Queue) BootstrapOne(task TaskIFace) error {
	return q.manager.bootstrapOne(task)
}

// Bootstrap boot注册载入多个队列任务
//  @tasks 任务类实例指针切片
func (q *Queue) Bootstrap(tasks []TaskIFace) error {
	return q.manager.bootstrap(tasks)
}

// endregion

// region 队列消费端相关方法

// Start 守护进程启动队列消费者
func (q *Queue) Start() error {
	// should continue process
	return q.manager.start()
}

// ShutDown graceful shut down
func (q *Queue) ShutDown(ctx context.Context) error {
	// graceful shutdown queue worker
	return q.manager.shutDown(ctx)
}

// endregion

// region 投递任务相关方法

// Dispatch 投递一个队列Job任务
func (q *Queue) Dispatch(task TaskIFace, payload interface{}) error {
	queuePayload, err := q.marshalPayload(task, payload)
	if nil != err {
		return fmt.Errorf("queue %s job param marshal failed: %s", task.Name(), err.Error())
	}

	return q.queue.Push(task.Name(), queuePayload)
}

// DelayAt 投递一个指定的将来时刻执行的延迟队列Job任务
func (q *Queue) DelayAt(task TaskIFace, payload interface{}, delay time.Time) error {
	queuePayload, err := q.marshalPayload(task, payload)
	if nil != err {
		return fmt.Errorf("queue %s job param marshal failed: %s", task.Name(), err.Error())
	}

	return q.queue.LaterAt(task.Name(), delay, queuePayload)
}

// Delay 投递一个指定延迟时长的延迟队列Job任务
func (q *Queue) Delay(task TaskIFace, payload interface{}, duration time.Duration) error {
	queuePayload, err := q.marshalPayload(task, payload)
	if nil != err {
		return fmt.Errorf("queue %s job param marshal failed: %s", task.Name(), err.Error())
	}

	return q.queue.Later(task.Name(), duration, queuePayload)
}

// DispatchByName 按任务name投递一个队列Job任务
//  - 投递一个异步立即执行的任务
//  - 重要:使用该方法则意味着投递任务之前必须bootstrap任务类，新项目请尽量使用Dispatch方法
func (q *Queue) DispatchByName(name string, payload interface{}) error {
	task, exist := q.manager.tasks[name]
	if !exist {
		return fmt.Errorf("queue %s do not bootstrap", name)
	}

	return q.Dispatch(task, payload)
}

// DelayAtByName 按任务name投递一个延迟队列Job任务
//  - 投递一个异步延迟执行的任务
//  - 重要提示:使用该方法则意味着投递任务之前必须bootstrap任务类，新项目请尽量使用DelayAt方法
func (q *Queue) DelayAtByName(name string, payload interface{}, delay time.Time) error {
	task, exist := q.manager.tasks[name]
	if !exist {
		return fmt.Errorf("queue %s do not bootstrap", name)
	}

	return q.DelayAt(task, payload, delay)
}

// DelayByName 按任务name投递一个将来时刻执行的延迟队列Job任务
//  - 投递一个异步延迟执行的任务
//  - 重要提示:使用该方法则意味着投递任务之前必须bootstrap任务类，新项目请尽量使用Delay方法
func (q *Queue) DelayByName(name string, payload interface{}, duration time.Duration) error {
	task, exist := q.manager.tasks[name]
	if !exist {
		return fmt.Errorf("queue %s do not bootstrap", name)
	}

	return q.Delay(task, payload, duration)
}

// Size 获取指定队列当前长度
func (q *Queue) Size(task TaskIFace) int64 {
	if _, exist := q.manager.tasks[task.Name()]; !exist {
		// 确保队列任务以注册
		return 0
	}
	return q.queue.Size(task.Name())
}

// SetHighPriorityTask 指定高优先级的Job任务，多次调用可以设置多个高优先级Job任务
//   - ① 当队列消费者消费速度过慢，任务堆积时被指定的高优先级Job将尽量保障优先执行
//   - ② 虽然此处可以指定队列job的高优先级执行，但也不保障待执行任务过多堆积时优先级任务一定会被执行，所以高优先级Job不要指定的过多
func (q *Queue) SetHighPriorityTask(task TaskIFace) error {
	return q.manager.setPriorityTask(task)
}

// SetAllowTasks 指定可以运行的任务
func (q *Queue) SetAllowTasks(taskNames ...string) {
	for _, name := range taskNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		q.logger.Info("queue set-allow-task", "taskName", name)
		q.manager.allowTasks[name] = struct{}{}
	}
}

// SetExcludeTasks 指定不可运行的任务
func (q *Queue) SetExcludeTasks(taskNames ...string) {
	for _, name := range taskNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		q.logger.Info("queue set-exclude-task", "taskName", name)
		q.manager.excludeTasks[name] = struct{}{}
	}
}

// endregion
