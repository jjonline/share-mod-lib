package guzzle

// 请求trace节点
const (
	GetConn              = "GetConn"
	DNSStart             = "DNSStart"
	DNSDone              = "DNSDone"
	ConnectStart         = "ConnectStart"
	ConnectDone          = "ConnectDone"
	TLSHandshakeStart    = "TLSHandshakeStart"
	TLSHandshakeDone     = "TLSHandshakeDone"
	GotConn              = "GotConn"
	WroteHeaders         = "WroteHeaders"
	WroteRequest         = "WroteRequest"
	GotFirstResponseByte = "GotFirstResponseByte"
)
