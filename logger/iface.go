package logger

// InterFace 通用logger接口定义
type InterFace interface {
	Info(msg string)
	Debug(msg string)
	Error(msg string)
}
