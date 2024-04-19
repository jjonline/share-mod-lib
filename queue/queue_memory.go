package queue

import (
	"container/list"
	"sync"
	"time"
)

/*
 * @Time   : 2021/1/16 上午11:20
 * @Email  : jingjing.yang@tvb.com
 */

// itemValue 延迟map实现、原生链表 实体结构
type itemValue struct {
	Payload Payload // job参数载体
	TimeAt  int64   // 承载延迟任务的执行时刻时间戳，非延迟任务值为0
}

// memoryQueue 基于memory实现的队列
// implement QueueIFace
type memoryQueue struct {
	queueBasic
	list     map[string]*list.List            // 原生链表模拟queue队列
	delayed  map[string]map[string]*itemValue // 使用map模拟延迟队列
	reserved map[string]map[string]*itemValue // 使用map模拟延迟队列
	lock     sync.Mutex
}

func (m *memoryQueue) Size(queue string) (size int64) {
	m.lazyInit(queue)

	return int64(m.list[queue].Len() + len(m.delayed[queue]) + len(m.reserved[queue]))
}

func (m *memoryQueue) Push(queue string, payload interface{}) (err error) {
	var originPayload Payload
	if err = m.unmarshalPayload(payload.([]byte), &originPayload); err != nil {
		return err
	}

	m.lazyInit(queue)

	item := &itemValue{
		Payload: originPayload,
		TimeAt:  0,
	}
	m.list[queue].PushBack(item)

	return nil
}

func (m *memoryQueue) Later(queue string, durationTo time.Duration, payload interface{}) (err error) {
	return m.LaterAt(queue, time.Now().Add(durationTo), payload)
}

func (m *memoryQueue) LaterAt(queue string, timeAt time.Time, payload interface{}) (err error) {
	var originPayload Payload
	if err = m.unmarshalPayload(payload.([]byte), &originPayload); err != nil {
		return err
	}

	m.lazyInit(queue)

	item := &itemValue{
		Payload: originPayload,
		TimeAt:  timeAt.Unix(),
	}

	// set to map
	m.delayed[queue][originPayload.ID] = item

	return nil
}

func (m *memoryQueue) Pop(queue string) (job JobIFace, exist bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	now := time.Now()
	// step1、调度延迟任务
	if m.delayed[queue] != nil {
		m.lazyInit(queue) // 延迟队列已初始化，但是保留队列可能未初始化
		// 迭代每个延迟job
		for id, item := range m.delayed[queue] {
			if item.TimeAt <= now.Unix() {
				// 执行时刻已到，将延迟任务丢到list
				itemV := &itemValue{
					Payload: item.Payload,
					TimeAt:  0,
				}

				// delete from delay map
				delete(m.delayed[queue], id)

				// push to list
				m.list[queue].PushBack(itemV)
			}
		}
	}

	// step2、处理保留重试任务
	if m.reserved[queue] != nil {
		m.lazyInit(queue) // 延迟队列已初始化，但是保留队列可能未初始化
		// 迭代每个延迟job
		for id, item := range m.reserved[queue] {
			if item.TimeAt <= now.Unix() {
				// 执行超时时刻已到，将延迟任务丢到list
				itemV := &itemValue{
					Payload: item.Payload,
					TimeAt:  0,
				}

				// delete from reserved map
				delete(m.reserved[queue], id)

				// push to list
				m.list[queue].PushBack(itemV)
			}
		}
	}

	// step3、调度list尝试执行
	if m.list[queue] == nil {
		return nil, false
	}

	// pop取出
	itemV := m.list[queue].Front()
	if itemV == nil {
		return nil, false
	}

	// 清理值
	m.list[queue].Remove(itemV)

	// 转义Payload初始化job
	node := *itemV.Value.(*itemValue)
	payload := node.Payload // value copy

	// 设置任务当前尝试次数和超时时刻等
	node.TimeAt = now.Add(time.Duration(node.Payload.Timeout) * time.Second).Unix()
	node.Payload.Attempts += 1
	if node.Payload.PopTime <= 0 {
		node.Payload.PopTime = now.Unix()
	}

	// set reserved
	m.reserved[queue][node.Payload.ID] = &node

	// 转换值构造job
	return &JobMemory{
		reserved:    m.reserved,
		delayed:     m.delayed,
		reservedJob: node.Payload,
		jobProperty: jobProperty{
			handler:    m,
			name:       queue,
			job:        "", // no useful
			reserved:   "", // no useful
			payload:    &payload,
			isReleased: false,
			isDeleted:  false,
			hasFailed:  false,
			popTime:    time.Unix(node.Payload.PopTime, 0),
			timeout:    time.Duration(payload.Timeout) * time.Second,
			timeoutAt:  now.Add(time.Duration(payload.Timeout) * time.Second),
		},
	}, true
}

func (m *memoryQueue) SetConnection(connection interface{}) (err error) {
	// no code
	return nil
}

func (m *memoryQueue) GetConnection() (connection interface{}, err error) {
	// no code
	return nil, nil
}

func (m *memoryQueue) lazyInit(queue string) {
	// lazy init map
	if m.list == nil {
		m.list = make(map[string]*list.List)
	}
	if m.reserved == nil {
		m.reserved = make(map[string]map[string]*itemValue)
	}
	if m.delayed == nil {
		m.delayed = make(map[string]map[string]*itemValue)
	}

	// lazy init map item
	if _, exist := m.list[queue]; !exist {
		m.list[queue] = list.New()
	}
	if _, exist := m.reserved[queue]; !exist {
		m.reserved[queue] = make(map[string]*itemValue)
	}
	if _, exist := m.delayed[queue]; !exist {
		m.delayed[queue] = make(map[string]*itemValue)
	}
}
