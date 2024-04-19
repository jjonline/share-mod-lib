package queue

import (
	"encoding/json"
	"github.com/google/uuid"
	"strconv"
	"time"
)

/*
 * @Time   : 2021/1/21 下午10:10
 * @Email  : jingjing.yang@tvb.com
 */

// FakeUniqueID 生成一个V4版本的uuid字符串，生成失败返回时间戳纳秒
// UUID单机足以保障唯一，生成失败场景下纳秒时间戳也可以一定程度上保障单机唯一
func FakeUniqueID() string {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return UUID.String()
}

// IFaceToString interface类型转string
func IFaceToString(value interface{}) string {
	var key string
	if value == nil {
		return key
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		key = strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		key = strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		key = strconv.Itoa(it)
	case uint:
		it := value.(uint)
		key = strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		key = strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		key = strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		key = strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		key = strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		key = strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		key = strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		key = strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		key = strconv.FormatUint(it, 10)
	case string:
		key = value.(string)
	case []byte:
		key = string(value.([]byte))
	default:
		newValue, _ := json.Marshal(value)
		key = string(newValue)
	}

	return key
}
