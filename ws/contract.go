package ws

import (
	"net/http"
	"sync/atomic"
)

// Logger 日志接口定义
type Logger interface {
	// Debug debug级别输出的日志
	//   - msg 日志消息文本描述
	//   - keyValue 按顺序一个key一个value，len(keyValue)一定是偶数<注意0也是偶数>
	Debug(msg string, keyValue ...string)
	// Info info级别输出的日志
	//   - msg 日志消息文本描述
	//   - keyValue 按顺序一个key一个value，len(keyValue)一定是偶数<注意0也是偶数>
	Info(msg string, keyValue ...string)
	// Warn warn级别输出的日志
	//   - msg 日志消息文本描述
	//   - keyValue 按顺序一个key一个value，len(keyValue)一定是偶数<注意0也是偶数>
	Warn(msg string, keyValue ...string)
	// Error error级别输出的日志
	//   - msg 日志消息文本描述
	//   - keyValue 按顺序一个key一个value，len(keyValue)一定是偶数<注意0也是偶数>
	Error(msg string, keyValue ...string)
}

type atomicBool int32

func (b *atomicBool) isTrue() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue()     { atomic.StoreInt32((*int32)(b), 1) }
func (b *atomicBool) setFalse()    { atomic.StoreInt32((*int32)(b), 0) }

type UserInfo struct {
	Uid         string
	Lang        string // 语言：tc-繁体，en-英文
	UserToken   string // 用户token
	DeviceType  string // Device type(app h5 stb)
	ClientToken string // client token（仅app可获取到，h5为空）
	MzToken     string // mz token
}

// 认证函数
type authFunc func(r *http.Request) (UserInfo, error)

// 事件处理器
type eventHandler func(client *Client, msg Request)

// 接收到客户端消息hook：可用于保存消息记录
type messageRequestHookFunc func(Request)

// 发送消息给客户端hook：可用于保存消息记录
type messageResponseHookFunc func(Response)

// 读取连接的消息
type connMessage struct {
	id          int64
	messageType int
	message     []byte
}

// pong回复
type pongPayload struct {
	Event string `json:"event"`
}

type offlinePayload struct {
	Fd     string `json:"fd"`
	Remark string `json:"remark"`
}

// Request 接收用户端消息结构体
type Request struct {
	ID        int64  `json:"id"`         // 消息id，ws服务器生成（微秒时间戳）
	From      string `json:"from"`       // 发送人：Server or 用户boss_id
	To        string `json:"to"`         // 接收人：Server or 用户boss_id
	MessageID string `json:"message_id"` // Client message id
	Device    string `json:"device"`     // Device (app h5 stb)
	Event     string `json:"event"`      // 事件
	Payload   string `json:"payload"`    // 请求参数 or 响应结果：json字符串
	SendTime  int64  `json:"send_time"`  // 发送时间戳
	Ignore    bool   `json:"-"`          // 是否忽略（不写入消息表）
}

// Response 响应用户消息结构体
type Response struct {
	ID       int64       `json:"id"`        // 消息id，ws服务器生成（微秒时间戳）
	From     string      `json:"from"`      // 发送人：Server or 用户boss_id
	To       string      `json:"to"`        // 接收人：Server or 用户boss_id
	Device   string      `json:"device"`    // Device (app h5 stb)
	Event    string      `json:"event"`     // 事件
	Payload  interface{} `json:"payload"`   // 请求参数 or 响应结果：json字符串
	SendTime int64       `json:"send_time"` // 发送时间戳
	Ignore   bool        `json:"-"`         // 是否忽略（不写入消息表）
}

// ServerInfo 服务器详情
type ServerInfo struct {
	AppID          string `json:"app_id"`           // appid
	ServerID       string `json:"server_id"`        // 服务器id
	LastActiveTime int64  `json:"last_active_time"` // 服务器最后存活时间
	ConnectionNum  int64  `json:"connection_num"`   // 客户端连接数量
}

// ConnectInfo 连接信息
type ConnectInfo struct {
	Fd             string `json:"fd"`               // fd连接id
	Uid            string `json:"uid"`              // 用户id
	ConnectTime    int64  `json:"connect_time"`     // 连接时间
	LastActiveTime int64  `json:"last_active_time"` // 最后存活时间（客户端发送消息、发送心跳会更新）
}

type ConnectionResult struct {
	Total       int64         `json:"total"`
	Cursor      int64         `json:"cursor"`
	Connections []ConnectInfo `json:"connections"`
}
