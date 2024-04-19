package ws

import "time"

const (
	shutdownPollIntervalMax  = 500 * time.Millisecond // 优雅关闭进程最大重复尝试间隔时长
	heartbeatTimeoutDuration = time.Second * 100      // 连接心跳超时时间，强制关闭连接

	ServerFd         = "server"
	ServerList       = "ws:server_list:%s"        // 记录ws服务器列表：ws:server_list:{app_id}
	ServerMsgPool    = "ws:server_msg_pool:%s"    // 记录服务器消息列表：ws:server_msg_pool:{server_id} => [fd_or_server:content]
	ServerClientList = "ws:server_client_list:%s" // 记录服务器的连接记录hash表：ws:server_conn_list:{server_id} => uid => fd:connect_time:last_active_time
	ClientInfoKey    = "ws:%s:client:%s"          // 记录集群用户连接信息（有效期十分钟，需要在心跳时不断续期）：ws:{appid}:client:{uid} => server_id:fd

	EventConnect    = "connect" // 上线通知（触发消息事件）
	EventOffline    = "offline" // 下线通知（触发消息事件）
	EventMsgConfirm = "confirm" // 消息确认（目标：客户端）
	EventPing       = "ping"    // ping（目标：服务器）
	EventPong       = "pong"    // pong（目标：客户端）

	LangTc = "tc" // 繁体
	LangEn = "en" // 英文
)
