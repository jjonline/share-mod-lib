package queue

import (
	"encoding/json"
)

/*
 * @Time   : 2021/1/20 下午10:10
 * @Email  : jingjing.yang@tvb.com
 */

// queueBasic 队列基础公用方法
type queueBasic struct{}

// region 获取队列相关名称私有方法

// name 获取队列名称
func (r *queueBasic) name(queue string) string {
	return queue
}

// reservedName 获取队列执行中zSet名称
func (r *queueBasic) reservedName(queue string) string {
	return queue + ":reserved"
}

// delayedName 获取队列延迟zSet名称
func (r *queueBasic) delayedName(queue string) string {
	return queue + ":delayed"
}

// marshalPayload 初始化创建生成队列内部存储的payload字符串
// @task	  队列任务类实例
// @taskParam 队列job参数
// @ID	      队列job编号ID（延迟队列）
func (r *queueBasic) marshalPayload(task TaskIFace, taskParam interface{}) ([]byte, error) {
	return json.Marshal(Payload{
		Name:          task.Name(),
		ID:            FakeUniqueID(),
		MaxTries:      task.MaxTries(),
		RetryInterval: task.RetryInterval(),
		Attempts:      0,
		Payload:       []byte(IFaceToString(taskParam)),
		PopTime:       0,                               // 首次被取出开始执行的时间戳，取出的时候才去设置
		Timeout:       int64(task.Timeout().Seconds()), // 最大执行秒数
		TimeoutAt:     0,                               // 超时时刻，被执行时刻才会去设置
	})
}

// unmarshalPayload 解析生成队列内部存储的payload字符串为struct
// @payload 队列内部存储的payload字符串
func (r *queueBasic) unmarshalPayload(payload []byte, result *Payload) error {
	return json.Unmarshal(payload, result)
}

// endregion
