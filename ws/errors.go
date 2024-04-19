package ws

import "errors"

var (
	ErrWsClosed            = errors.New("ws.error.websocket.service.closed")
	ErrWsClientClosed      = errors.New("ws.error.websocket.client.closed")
	ErrWsClientError       = errors.New("ws.error.websocket.client.error")
	ErrWsClientDoNotExist  = errors.New("ws.error.websocket.client.do.not.exist")
	ErrWsClientInfoError   = errors.New("ws.error.websocket.client.info.error")
	ErrWsUserNotLoginError = errors.New("ws.error.websocket.user.not_login.error")
)
