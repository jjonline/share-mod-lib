package ws

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
	"strconv"
	"time"
)

// Client ws单个客户端抽象
type Client struct {
	fd               string            // ws分配的全局唯一ID
	conn             *websocket.Conn   // ws网络连接层抽象
	server           *Server           // ws-manager指针，便于与manager交互
	userInfo         UserInfo          // 用户信息
	connectTime      time.Time         // 连接时间
	lastActiveTime   time.Time         // ws客户端最近1次活动<心跳、交互>时间
	isClosed         atomicBool        // 是否已关闭
	receiveMessageCh chan *connMessage // conn接收消息channel
	sendMessageCh    chan *connMessage // conn发送消息channel
}

func (c *Client) GetFd() string {
	return c.fd
}

func (c *Client) GetUserInfo() UserInfo {
	return c.userInfo
}

func (c *Client) GetUid() string {
	return c.userInfo.Uid
}

func (c *Client) GetLang() string {
	return c.userInfo.Lang
}

func (c *Client) GetUserToken() string {
	return c.userInfo.UserToken
}

func (c *Client) GetUserDevice() string {
	return c.userInfo.DeviceType
}

func (c *Client) GetClientToken() string {
	return c.userInfo.ClientToken
}

func (c *Client) GetMzToken() string {
	return c.userInfo.MzToken
}

func (c *Client) GetConnectTime() time.Time {
	return c.connectTime
}

func (c *Client) GetLastActiveTime() time.Time {
	return c.lastActiveTime
}

// tick 协程处理消息收发
func (c *Client) tick() {
	c.server.logger.Info("websocket service client start tick",
		"appid", c.server.appid, "server_id", c.server.id, "fd", c.fd, "uid", c.GetUid())

	// 触发连接事件（通知服务端）
	go c.emitConnect()

	// 接收消息
	go c.receiveMessage()

	// 发送消息
	go c.sendMessage()

	for {
		if c.isClosed.isTrue() {
			break
		}

		mt, message, err := c.conn.ReadMessage()
		if err != nil {
			_ = c.close("read message err:" + err.Error())
			c.server.logger.Info("websocket client read message error",
				"appid", c.server.appid, "server_id", c.server.id, "fd", c.fd, "uid", c.GetUid(), "error", err.Error())
			break
		}

		// 更新最新活跃时间
		c.lastActiveTime = time.Now()

		// 更新用户连接信息有效期
		c.server.updateClientTTL(c.GetUid())

		// 处理消息
		c.receiveMessageCh <- &connMessage{id: time.Now().UnixMicro(), messageType: mt, message: message}
	}
}

func (c *Client) receiveMessage() {
	for msg := range c.receiveMessageCh {
		if c.isClosed.isTrue() {
			return
		}

		c.msgHandler(msg.id, msg.messageType, msg.message)
	}
}

// websocket connect write方法不支持并发，需要使用channel
func (c *Client) sendMessage() {
	for msg := range c.sendMessageCh {
		c.server.logger.Debug("websocket send message to client",
			"appid", c.server.appid, "server_id", c.server.id, "fd", c.fd, "uid", c.GetUid(),
			"message_type", strconv.Itoa(msg.messageType), "data", string(msg.message))

		if c.isClosed.isTrue() {
			return
		}

		if err := c.conn.WriteMessage(msg.messageType, msg.message); err != nil {
			c.server.logger.Debug("websocket send message to client err",
				"appid", c.server.appid, "server_id", c.server.id, "fd", c.fd, "uid", c.GetUid(),
				"message_type", strconv.Itoa(msg.messageType), "data", string(msg.message), "err", err.Error())
		}
	}
}

// msgHandler 消息处理器
func (c *Client) msgHandler(messageID int64, messageType int, data []byte) {
	c.server.logger.Debug("websocket receive client message",
		"appid", c.server.appid, "server_id", c.server.id, "fd", c.fd, "uid", c.GetUid(),
		"message_id", strconv.FormatInt(messageID, 10), "message_type", strconv.Itoa(messageType), "data", string(data))

	switch messageType {
	case websocket.TextMessage:
		// 解析消息
		replyID, msg := c.parseMessage(messageID, data)

		// 回复心跳消息
		if msg.Event == EventPing {
			_ = c.pong()
			return
		}

		// 触发消息钩子
		go c.server.emitMessageRequestHook(msg)

		// 回复确认消息
		_ = c.replyConfirmMessage(replyID, msg.ID)

		// 触发消息事件
		go c.server.emitEvent(msg.Event, c, msg)
	case websocket.BinaryMessage:
		// 不支持
	case websocket.CloseMessage:
		// 关闭连接，客户端关闭时会先出现消息读取错误，一般不会触发到此处
		_ = c.close("websocket.CloseMessage")
	case websocket.PingMessage:
		// 不支持
	}

	return
}

// parseMessage 解析消息：将payload（json对象）解析为json字符串
func (c *Client) parseMessage(messageID int64, data []byte) (replyID string, msg Request) {
	result := gjson.ParseBytes(data)

	replyID = result.Get("id").String()
	msg = Request{
		ID:        messageID,
		From:      c.GetUid(),
		To:        ServerFd,
		MessageID: replyID,
		Device:    c.GetUserDevice(),
		Event:     result.Get("event").String(),
		Payload:   result.Get("payload").String(),
		SendTime:  time.Now().Unix(),
	}
	return
}

// replyConfirmMessage 回复确认消息
// replyID   收到客户端的临时消息id，原样返回，通知客户端此消息已收到
// messageID 此消息为数据库的消息id，返回给客户端进行保存，作为消息id，替换临时消息id；
//
//	客户端获取历史消息时，需要使用消息id来标记开始位置，消息id仅某用户下唯一，非全局唯一
func (c *Client) replyConfirmMessage(replyID string, messageID int64) (err error) {
	if replyID == "" {
		return
	}

	// 确认消息
	type confirmMessage struct {
		ID string `json:"id"` // 客户端请求的消息id，原样返回
	}

	msg := Response{
		ID:       messageID,
		From:     ServerFd,
		To:       c.GetUid(),
		Device:   c.GetUserDevice(),
		Event:    EventMsgConfirm,
		Payload:  confirmMessage{ID: replyID}, // 客户端请求的消息id，原样返回，通知客户端此消息已收到
		SendTime: time.Now().Unix(),
	}

	b, _ := json.Marshal(msg)
	return c.write(websocket.TextMessage, b)
}

func (c *Client) pong() (err error) {
	b, _ := json.Marshal(pongPayload{Event: EventPong})
	return c.write(websocket.TextMessage, b)
}

func (c *Client) write(messageType int, content []byte) (err error) {
	if c.isClosed.isTrue() {
		return
	}

	c.sendMessageCh <- &connMessage{
		id:          0,
		messageType: messageType,
		message:     content,
	}
	return
}

// 关闭连接
func (c *Client) close(remark string) (err error) {
	c.server.logger.Info("websocket client close",
		"appid", c.server.appid, "server_id", c.server.id, "fd", c.fd, "uid", c.GetUid(),
		"remark", remark)

	if !c.isClosed.isTrue() {
		c.isClosed.setTrue()                    // 设置连接关闭标识
		_ = c.conn.Close()                      // 关闭连接
		close(c.receiveMessageCh)               // 关闭接收消息通道
		close(c.sendMessageCh)                  // 关闭发送消息通道
		go c.emitOffline(remark)                // 触发下线事件（通知服务端）
		c.server.deleteClient(c.GetUid(), c.fd) // 从服务器删除client
	}
	return
}

// SendMessage 向用户发送信息
func (c *Client) SendMessage(event string, payload interface{}) (err error) {
	msg := Response{
		ID:       time.Now().UnixMicro(),
		From:     ServerFd,
		To:       c.GetUid(),
		Device:   c.GetUserDevice(),
		Event:    event,
		Payload:  payload,
		SendTime: time.Now().Unix(),
	}

	go c.server.emitMessageResponseHook(msg)

	if c.isClosed.isTrue() {
		return ErrWsClientClosed
	}

	b, _ := json.Marshal(msg)
	return c.write(websocket.TextMessage, b)
}

// 触发连接事件（通知服务端）
func (c *Client) emitConnect() {
	c.server.emitEvent(EventConnect, c, Request{
		ID:       time.Now().UnixMicro(),
		From:     ServerFd,
		To:       ServerFd,
		Device:   c.GetUserDevice(),
		Event:    EventConnect,
		Payload:  "",
		SendTime: time.Now().Unix(),
		Ignore:   true,
	})
}

// 触发下线事件（通知服务端）
func (c *Client) emitOffline(remark string) {
	c.server.emitEvent(EventOffline, c, Request{
		ID:       time.Now().UnixMicro(),
		From:     ServerFd,
		To:       ServerFd,
		Device:   c.GetUserDevice(),
		Event:    EventOffline,
		Payload:  remark,
		SendTime: time.Now().Unix(),
		Ignore:   true,
	})
}
