package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	appid               string                   // 应用id
	id                  string                   // 服务器id
	shutdown            atomicBool               // 关闭标识
	ctx                 context.Context          // context
	redis               *redis.Client            // redis客户端
	clients             sync.Map                 // 保存当前服务器的客户端
	heartBeatTicker     *time.Ticker             // 客户端心跳检测ticker
	messagePoolTicker   *time.Ticker             // 消息池ticker
	selfCheckingTicker  *time.Ticker             // 服务自检ticker
	upgrader            websocket.Upgrader       // upgrader
	authFunc            authFunc                 // 鉴权func
	eventHandlers       sync.Map                 // 保存注册的事件处理器
	acceptClientCh      chan *Client             // 客户端连接channel
	logger              Logger                   // logger
	messageRequestHook  *messageRequestHookFunc  // 接收到客户端消息hook：可用于保存消息记录
	messageResponseHook *messageResponseHookFunc // 发送消息给客户端hook：可用于保存消息记录
}

func NewServer(appid string, subProtocols []string, redisCli *redis.Client, logger Logger) *Server {
	return &Server{
		appid:              appid,
		id:                 strconv.FormatInt(time.Now().UnixNano(), 10),
		ctx:                context.TODO(),
		redis:              redisCli,
		clients:            sync.Map{},
		heartBeatTicker:    time.NewTicker(time.Second * 10),
		messagePoolTicker:  time.NewTicker(time.Millisecond * 100),
		selfCheckingTicker: time.NewTicker(time.Second * 30),
		acceptClientCh:     make(chan *Client, 5),
		logger:             logger,
		upgrader: websocket.Upgrader{
			HandshakeTimeout:  5 * time.Second,
			ReadBufferSize:    0,
			WriteBufferSize:   0,
			WriteBufferPool:   nil,
			Subprotocols:      subProtocols, // 注册ws-子协议名称
			Error:             nil,
			EnableCompression: false,
			CheckOrigin: func(r *http.Request) bool {
				return true // 使用Subprotocols必须返回true
			},
		},
	}
}

// Serve 开始serve，阻塞
func (s *Server) Serve() error {
	if s.shutdown.isTrue() {
		return ErrWsClosed
	}

	s.logger.Info("websocket service start")

	s.register()
	go s.tickMessagePool()
	go s.checkHeartbeatTimeout()
	go s.selfChecking()
	s.accept()

	return nil
}

// Handler http句柄注册器：标准net/http包下 http.HandlerFunc 用于将路由注册到websocket协议
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	// 处于关闭中状态时不再接收新请求
	if s.shutdown.isTrue() {
		w.Header().Set("Sec-Websocket-Version", "13")
		http.Error(w, "Websocket "+http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// 鉴权、获取用户信息
	userInfo, err := s.authFunc(r)
	if err != nil {
		s.logger.Error("websocket client auth failed",
			"appid", s.appid, "server_id", s.id, "header", r.Header.Get("Sec-Websocket-Protocol"), "err", err.Error())

		w.Header().Set("Sec-Websocket-Version", "13")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket client upgrader failed",
			"appid", s.appid, "server_id", s.id, "header", r.Header.Get("Sec-Websocket-Protocol"), "err", err.Error())

		w.Header().Set("Sec-Websocket-Version", "13")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// 客户端语言
	if userInfo.Lang == "" {
		userInfo.Lang = LangTc
	}

	client := &Client{
		server:           s,
		fd:               s.fakeUUID(),
		conn:             conn,
		userInfo:         userInfo,
		connectTime:      time.Now(),
		lastActiveTime:   time.Now(),
		receiveMessageCh: make(chan *connMessage, 10),
		sendMessageCh:    make(chan *connMessage, 10),
	}
	s.acceptClientCh <- client

	s.logger.Info("websocket service start accepted client",
		"appid", s.appid, "server_id", s.id, "fd", client.fd, "uid", client.GetUid())
}

// Shutdown 停止服务
func (s *Server) Shutdown(ctx context.Context) (err error) {
	if s.shutdown.isTrue() {
		return ErrWsClosed
	}

	s.logger.Info("websocket service shutting down", "appid", s.appid, "server_id", s.id)

	s.shutdown.setTrue()
	s.stopAccept()
	s.stopHeartbeatCheck()
	s.stopMessagePoolTicker()
	s.closeAllClient()
	s.deleteAllClientConnectLog()
	s.stopSelfChecking()
	s.unregister()

	return s.waitShutdownFinish(ctx)
}

func (s *Server) waitShutdownFinish(ctx context.Context) (err error) {
	s.logger.Info("websocket service waiting shutdown finish", "appid", s.appid, "server_id", s.id)

	// 优雅关闭等待时长逐步递增实现
	pollIntervalBase := time.Millisecond
	nextPollInterval := func() time.Duration {
		// Add 10% jitter.
		interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		// Double and clamp for next time.
		pollIntervalBase *= 2
		if pollIntervalBase > shutdownPollIntervalMax {
			pollIntervalBase = shutdownPollIntervalMax
		}
		return interval
	}

	timer := time.NewTimer(nextPollInterval())
	defer timer.Stop()
	for {
		// 检测到所有client已经关闭完毕，shutdown完成
		clientNum := 0
		s.clients.Range(func(key, value any) bool {
			clientNum += 1
			return false
		})
		if clientNum <= 0 {
			s.logger.Info("websocket service shutdown finish", "appid", s.appid, "server_id", s.id)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(nextPollInterval())
		}
	}
}

// 注册
func (s *Server) register() {
	s.logger.Info("websocket service register", "appid", s.appid, "server_id", s.id)
	s.redis.HSet(s.ctx, fmt.Sprintf(ServerList, s.appid), s.id, time.Now().Unix())
}

func (s *Server) unregister() {
	s.logger.Info("websocket service unregister", "appid", s.appid, "server_id", s.id)
	// 从服务器集群中删除
	s.redis.HDel(s.ctx, fmt.Sprintf(ServerList, s.appid), s.id)
	// 删除该服务器对应的消息池
	s.redis.Del(s.ctx, fmt.Sprintf(ServerMsgPool, s.id))
}

// GetServerList 获取集群内服务器列表（相同appid）：服务器id => 服务器最后存活时间
func (s *Server) GetServerList() (servers []ServerInfo, err error) {
	servers = make([]ServerInfo, 0)

	result, err := s.redis.HGetAll(s.ctx, fmt.Sprintf(ServerList, s.appid)).Result()
	if err != nil {
		return
	}

	for serverID, lastActiveTime := range result {
		connectNum, _ := s.redis.HLen(s.ctx, fmt.Sprintf(ServerClientList, serverID)).Result()
		servers = append(servers, ServerInfo{
			AppID:          s.appid,
			ServerID:       serverID,
			LastActiveTime: cast.ToInt64(lastActiveTime),
			ConnectionNum:  connectNum,
		})
	}

	return
}

// 服务自检
func (s *Server) selfChecking() {
	defer s.selfCheckingTicker.Stop()

	s.logger.Info("websocket service start self-checking tick", "appid", s.appid, "server_id", s.id)

	check := func() {
		// 检测服务器列表，移除已掉线的服务器
		_ = s.removeOfflineServer()

		// 更新当前服务器存活时间
		s.redis.HSet(s.ctx, fmt.Sprintf(ServerList, s.appid), s.id, time.Now().Unix())

		// 续期：当前服务器消息池有效期
		s.redis.Expire(s.ctx, fmt.Sprintf(ServerMsgPool, s.id), time.Minute*10)

		// 续期：当前服务器连接记录有效期
		s.redis.Expire(s.ctx, fmt.Sprintf(ServerClientList, s.id), time.Minute*10)
	}

	check()

	for range s.selfCheckingTicker.C {
		s.logger.Info("websocket service self-checking", "appid", s.appid, "server_id", s.id)

		check()
	}
}

// 停止服务自检
func (s *Server) stopSelfChecking() {
	s.logger.Info("websocket service stop self-checking tick", "appid", s.appid, "server_id", s.id)
	s.selfCheckingTicker.Stop()
}

// 移除已掉线的服务器：超过5分钟未更新存活时间视为已掉线
func (s *Server) removeOfflineServer() (err error) {
	serverList, err := s.GetServerList()
	if err != nil {
		return
	}

	for _, server := range serverList {
		if server.LastActiveTime < time.Now().Add(-5*time.Minute).Unix() {
			s.logger.Info("websocket service remove offline server", "appid", s.appid, "server_id", server.ServerID)
			// 从服务器集群中删除
			s.redis.HDel(s.ctx, fmt.Sprintf(ServerList, s.appid), server.ServerID)
			// 删除该服务器对应的消息池
			s.redis.Del(s.ctx, fmt.Sprintf(ServerMsgPool, server.ServerID))
			// 删除该服务器的连接记录
			s.redis.Del(s.ctx, fmt.Sprintf(ServerClientList, server.ServerID))
		}
	}
	return
}

func (s *Server) tickMessagePool() {
	defer s.messagePoolTicker.Stop()

	s.logger.Info("websocket service start tick message pool", "appid", s.appid, "server_id", s.id)

	for range s.messagePoolTicker.C {
		result, err := s.redis.RPop(s.ctx, fmt.Sprintf(ServerMsgPool, s.id)).Result()
		if err != nil {
			continue
		}

		fd, message, found := strings.Cut(result, ":")
		if !found {
			continue
		}

		// 发给服务器的消息
		if fd == ServerFd {
			go s.serverMsgHandler([]byte(message))
		} else {
			go s.send(fd, []byte(message))
		}
	}
}

// 发送消息
func (s *Server) send(fd string, message []byte) (err error) {
	client, err := s.getClientByFd(fd)
	if err != nil {
		return
	}
	return client.write(websocket.TextMessage, message)
}

func (s *Server) stopMessagePoolTicker() {
	s.logger.Info("websocket service stop message pool tick", "appid", s.appid, "server_id", s.id)

	s.messagePoolTicker.Stop()
	s.redis.Del(s.ctx, fmt.Sprintf(ServerMsgPool, s.id))
}

// 处理连接
func (s *Server) accept() {
	s.logger.Info("websocket service start accept client", "appid", s.appid, "server_id", s.id)

	for client := range s.acceptClientCh {
		s.registerClient(client)
		go client.tick()
	}
}

func (s *Server) stopAccept() {
	s.logger.Info("websocket service stop accept connect", "appid", s.appid, "server_id", s.id)

	close(s.acceptClientCh)
}

func (s *Server) checkHeartbeatTimeout() {
	defer s.heartBeatTicker.Stop()

	s.logger.Info("websocket service start check client heartbeat", "appid", s.appid, "server_id", s.id)

	for range s.heartBeatTicker.C {
		if s.shutdown.isTrue() {
			return
		}

		s.clients.Range(func(key, value any) bool {
			client, ok := value.(*Client)
			if !ok {
				s.clients.Delete(key)
				return true
			}

			// 心跳超时，关闭连接
			if time.Now().Sub(client.lastActiveTime) > heartbeatTimeoutDuration {
				_ = client.close("heartbeat timeout")
			}

			// 更新连接记录
			_ = s.writeClientConnectLog(client.fd, client.GetUid(), client.connectTime, client.lastActiveTime)

			return true
		})
	}
}

func (s *Server) stopHeartbeatCheck() {
	s.logger.Info("websocket service stop heartbeat check", "appid", s.appid, "server_id", s.id)

	s.heartBeatTicker.Stop()
}

// fakeUUID 生成一个V4版本的uuid字符串，生成失败返回纳秒时间戳字符串「注意：高并发场景可能会出现极低概率的重复」
func (s *Server) fakeUUID() string {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return "fd" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return UUID.String()
}

// RegisterAuthFunc 注册认证函数
func (s *Server) RegisterAuthFunc(f authFunc) {
	s.logger.Info("websocket service register auth func", "appid", s.appid, "server_id", s.id)

	s.authFunc = f
	return
}

// RegisterEvent 注册事件
func (s *Server) RegisterEvent(event string, f eventHandler) {
	s.logger.Info("websocket service register event",
		"appid", s.appid, "server_id", s.id, "event", event)

	s.eventHandlers.Store(event, &f)
}

// emitEvent 触发事件
func (s *Server) emitEvent(event string, client *Client, msg Request) {
	defer func() {
		if err := recover(); err != nil {
			s.logger.Error(fmt.Sprintf("websocket service emitEvent recover:%v", err))
		}
	}()

	if value, exists := s.eventHandlers.Load(event); event != "" && exists {
		s.logger.Debug("websocket service client emit event",
			"appid", s.appid, "server_id", s.id, "event", event)

		if handler, ok := value.(*eventHandler); ok {
			(*handler)(client, msg)
		}
	} else {
		// 未注册的事件
		s.logger.Debug("websocket service client emit unregistered event",
			"appid", s.appid, "server_id", s.id, "event", event)
	}
}

// ForceOffline 强制下线
func (s *Server) ForceOffline(uid, remark string) (err error) {
	serverID, fd, err := s.getClientInfoByUid(uid)
	if err != nil {
		return
	}

	if serverID == "" || fd == "" {
		return
	}

	s.logger.Info("websocket service send force offline message",
		"appid", s.appid, "server_id", s.id, "uid", uid, "remark", remark)

	// 发送强制下线消息
	msg := Response{
		ID:       time.Now().UnixMicro(),
		From:     ServerFd,
		To:       ServerFd,
		Event:    EventOffline,
		Payload:  offlinePayload{Fd: fd, Remark: remark},
		SendTime: time.Now().Unix(),
	}
	return s.dispatchMessage(serverID, ServerFd, msg)
}

// 系统消息处理
func (s *Server) serverMsgHandler(content []byte) (err error) {
	var resp Response
	if err = json.Unmarshal(content, &resp); err != nil {
		return
	}

	// payload转为byte，根据event解析
	payloadBytes, err := json.Marshal(resp.Payload)
	if err != nil {
		return
	}

	switch resp.Event {
	case EventOffline: // 强制下线
		var offlineMsg offlinePayload
		if err = json.Unmarshal(payloadBytes, &offlineMsg); err != nil {
			return
		}

		// 通知用户并强制下线
		_ = s.send(offlineMsg.Fd, content)
		time.Sleep(time.Second)
		err = s.closeClientByFd(offlineMsg.Fd, offlineMsg.Remark)
	}

	return
}

func (s *Server) registerClient(client *Client) {
	s.logger.Info("websocket service register client",
		"appid", s.appid, "server_id", s.id, "fd", client.fd, "uid", client.GetUid())

	// 强制下线用户已有连接
	_ = s.ForceOffline(client.GetUid(), "user login elsewhere, force offline")

	// 保存连接记录
	_ = s.writeClientConnectLog(client.fd, client.GetUid(), client.connectTime, client.lastActiveTime)

	// 接收加入新ws-client
	s.clients.Store(client.fd, client)

	// 用户连接关系: ws:{appid}:client:{uid} => server_id:fd
	s.redis.Set(s.ctx,
		fmt.Sprintf(ClientInfoKey, s.appid, client.GetUid()),
		fmt.Sprintf("%s:%s", s.id, client.fd),
		time.Minute*10)
}

// 获取用户连接信息
func (s *Server) getClientInfoByUid(uid string) (serverID, fd string, err error) {
	// 根据uid获取所在服务器&fd
	result, err := s.redis.Get(s.ctx, fmt.Sprintf(ClientInfoKey, s.appid, uid)).Result()
	if err != nil {
		return
	}

	serverID, fd, found := strings.Cut(result, ":")
	if result == "" || !found {
		err = ErrWsClientInfoError
		return
	}
	return
}

// 更新用户连接信息有效期
func (s *Server) updateClientTTL(uid string) {
	s.redis.Expire(s.ctx, fmt.Sprintf(ClientInfoKey, s.appid, uid), time.Minute*10)
}

// 清除用户连接信息
func (s *Server) deleteClient(uid, fd string) {
	s.clients.Delete(fd)

	// 删除连接记录
	_ = s.deleteClientConnectLog(uid)

	// 删除用户连接关系
	serverID, oldFd, _ := s.getClientInfoByUid(uid)
	if serverID == s.id && oldFd == fd {
		_, _ = s.redis.Del(s.ctx, fmt.Sprintf(ClientInfoKey, s.appid, uid)).Result()
	}
	return
}

// 关闭client
func (s *Server) closeClientByFd(fd, remark string) (err error) {
	client, err := s.getClientByFd(fd)
	if err != nil {
		return
	}
	return client.close(remark)
}

// 关闭所以连接
func (s *Server) closeAllClient() {
	s.logger.Info("websocket service close all client", "appid", s.appid, "server_id", s.id)

	s.clients.Range(func(key, value any) bool {
		if client, ok := value.(*Client); ok {
			_ = client.close("server shutdown")
		}
		return true
	})
}

func (s *Server) getClientByFd(fd string) (c *Client, err error) {
	value, ok := s.clients.Load(fd)
	if !ok {
		return nil, ErrWsClientDoNotExist
	}

	c, ok = value.(*Client)
	if !ok {
		return nil, ErrWsClientError
	}
	return
}

// GetServerConnections 根据serverID获取连接记录
// 注意事项：当总数大于512时，分页limit才会生效；返回的cursor大于0才有下一页
func (s *Server) GetServerConnections(serverID string, cursor, limit int64) (resp ConnectionResult, err error) {
	resp.Connections = make([]ConnectInfo, 0)

	if limit == 0 {
		limit = 10
	}

	fmt.Println(strings.Repeat("=", 20), cursor, limit)

	// 根据serverID获取连接列表：uid => fd:connect_time:last_active_time
	key := fmt.Sprintf(ServerClientList, serverID)

	// 获取总数
	resp.Total, err = s.redis.HLen(s.ctx, key).Result()
	if err != nil {
		return
	}

	result, nextCursor, err := s.redis.HScan(s.ctx, key, uint64(cursor), "", limit).Result()
	if err != nil {
		return
	}

	// 结果格式：[]string：key, value, key, value...
	for i, value := range result {
		if i%2 == 1 {
			connectInfo := s.parseClientConnectLog(value)
			connectInfo.Uid = result[i-1]
			resp.Connections = append(resp.Connections, connectInfo)
		}
	}

	resp.Cursor = int64(nextCursor)
	return
}

// 写入连接记录
func (s *Server) writeClientConnectLog(fd, uid string, connectTime, lastActiveTime time.Time) (err error) {
	key := fmt.Sprintf(ServerClientList, s.id)
	_, err = s.redis.HSet(s.ctx, key, uid, fmt.Sprintf("%s:%d:%d", fd, connectTime.Unix(), lastActiveTime.Unix())).Result()
	return
}

// 删除连接记录
func (s *Server) deleteClientConnectLog(uid string) (err error) {
	key := fmt.Sprintf(ServerClientList, s.id)
	_, err = s.redis.HDel(s.ctx, key, uid).Result()
	return
}

// 解析连接log信息：uid:connect_time:last_active_time
func (s *Server) parseClientConnectLog(str string) (info ConnectInfo) {
	arr := strings.Split(str, ":")
	if len(arr) == 3 {
		info.Fd = arr[0]
		info.ConnectTime = cast.ToInt64(arr[1])
		info.LastActiveTime = cast.ToInt64(arr[2])
	}
	return
}

// 删除全部连接记录
func (s *Server) deleteAllClientConnectLog() {
	s.logger.Info("websocket service delete client connect log", "appid", s.appid, "server_id", s.id)

	key := fmt.Sprintf(ServerClientList, s.id)
	_, _ = s.redis.Del(s.ctx, key).Result()
}

// SendMessage 向用户推送消息：将消息写入每个server对应的消息池
func (s *Server) SendMessage(uid string, event string, payload interface{}) (err error) {
	msg := Response{
		ID:       time.Now().UnixMicro(),
		From:     ServerFd,
		To:       uid,
		Event:    event,
		Payload:  payload,
		SendTime: time.Now().Unix(),
	}

	s.logger.Debug("websocket service send message to client",
		"appid", s.appid, "server_id", s.id, "uid", uid, "event", event)

	// 触发响应钩子（用户不在线也需要保存消息记录）
	go s.emitMessageResponseHook(msg)

	// 根据uid获取所在服务器&fd
	serverID, fd, err := s.getClientInfoByUid(uid)
	if err != nil {
		return err
	}

	if serverID == "" || fd == "" {
		return ErrWsUserNotLoginError
	}

	return s.dispatchMessage(serverID, fd, msg)
}

func (s *Server) dispatchMessage(serverID, fd string, msg Response) (err error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}

	s.logger.Debug("websocket service dispatch message to server",
		"appid", s.appid, "server_id", s.id, "target_server", serverID, "fd", fd)

	// 消息内容格式：fd:message
	_, err = s.redis.LPush(s.ctx, fmt.Sprintf(ServerMsgPool, serverID),
		fmt.Sprintf("%s:%s", fd, string(b))).Result()
	return
}

// RegisterMessageRequestHook 注册钩子：接收到客户端消息hook：可用于保存消息记录
func (s *Server) RegisterMessageRequestHook(f messageRequestHookFunc) {
	s.messageRequestHook = &f
}

// RegisterMessageResponseHook 注册钩子：接收到客户端消息hook：可用于保存消息记录
func (s *Server) RegisterMessageResponseHook(f messageResponseHookFunc) {
	s.messageResponseHook = &f
}

// 触发钩子
func (s *Server) emitMessageRequestHook(request Request) {
	defer func() {
		if err := recover(); err != nil {
			s.logger.Error(fmt.Sprintf("websocket service emitMessageRequestHook recover:%v", err))
		}
	}()

	if s.messageRequestHook != nil {
		(*s.messageRequestHook)(request)
	}
}

// 触发钩子
func (s *Server) emitMessageResponseHook(response Response) {
	defer func() {
		if err := recover(); err != nil {
			s.logger.Error(fmt.Sprintf("websocket service emitMessageResponseHook recover:%v", err))
		}
	}()

	if s.messageResponseHook != nil {
		(*s.messageResponseHook)(response)
	}
}
