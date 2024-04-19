package queue

import (
	"fmt"
	"time"
)

/*
 * @Time   : 2021/03/08 14:40
 * @Email  : jingjing.yang@tvb.com
 */

type JobMemory struct {
	basic       queueBasic
	delayed     map[string]map[string]*itemValue // 延迟map ref type
	reserved    map[string]map[string]*itemValue // 保留map ref type
	reservedJob Payload                          // 处理后的保留状态的job
	jobProperty
}

func (job *JobMemory) Release(delay int64) (err error) {
	job.isReleased = true

	if _, exist := job.reserved[job.GetName()]; !exist {
		return fmt.Errorf("queue %s do no exist", job.GetName())
	}

	if _, exist := job.reserved[job.GetName()][job.payload.ID]; !exist {
		return fmt.Errorf("queue %s do no exist this job, id=%s", job.GetName(), job.payload.ID)
	}

	if _, exist := job.delayed[job.GetName()]; !exist {
		return fmt.Errorf("queue %s do no exist", job.GetName())
	}

	// 从保留队列删除
	delete(job.reserved[job.GetName()], job.payload.ID)

	// 移动到延迟队列
	itemV := itemValue{
		Payload: job.reservedJob,
		TimeAt:  time.Now().Add(time.Duration(delay) * time.Second).Unix(),
	}
	job.delayed[job.GetName()][job.payload.ID] = &itemV

	return nil
}

func (job *JobMemory) Delete() (err error) {
	job.isDeleted = true

	if _, exist := job.reserved[job.GetName()]; !exist {
		return fmt.Errorf("queue %s do no exist", job.GetName())
	}

	if _, exist := job.reserved[job.GetName()][job.payload.ID]; !exist {
		return fmt.Errorf("queue %s do no exist this job, id=%s", job.GetName(), job.payload.ID)
	}

	// 从保留队列删除
	delete(job.reserved[job.GetName()], job.payload.ID)

	return nil
}

func (job *JobMemory) IsDeleted() (deleted bool) {
	return job.isDeleted
}

func (job *JobMemory) IsReleased() (released bool) {
	return job.isReleased
}

func (job *JobMemory) Attempts() (attempt int64) {
	return job.payload.Attempts + 1
}

func (job *JobMemory) PopTime() (time time.Time) {
	return job.popTime
}

// Timeout 任务超时时长
func (job *JobMemory) Timeout() (time time.Duration) {
	return job.jobProperty.timeout
}

// TimeoutAt 任务job执行超时的时刻
func (job *JobMemory) TimeoutAt() (time time.Time) {
	return job.jobProperty.timeoutAt
}

func (job *JobMemory) HasFailed() (hasFail bool) {
	return job.hasFailed
}

func (job *JobMemory) MarkAsFailed() {
	job.hasFailed = true
}

func (job *JobMemory) Failed(err error) {
	// no code
}

func (job *JobMemory) GetName() (queueName string) {
	return job.name
}

func (job *JobMemory) Queue() (queue QueueIFace) {
	return job.handler
}

func (job *JobMemory) Payload() (payload *Payload) {
	return job.payload
}
