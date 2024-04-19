package logger

import (
	"fmt"
	"go.uber.org/zap"
)

// Logger logger封装, 实现第三方库的日志接口
type Logger struct {
	recordField string // xxxRecord方法额外添加分门别类字段的名称
	Zap         *zap.Logger
}

// New 初始化单例logger
//   -- level 日志级别：debug、info、warning 等
//   -- path  文件形式的日志路径 or 标准输出 stderr
//   -- recordField  指定xxxRecord系列方法额外添加分门别类的方法的字段名称
func New(level, path, recordField string) *Logger {
	if recordField == "" {
		recordField = "module"
	}
	return &Logger{
		recordField: recordField,
		Zap:         newZap(level, path),
	}
}

func (l *Logger) Debug(msg string) {
	l.Zap.Debug(msg)
}
func (l *Logger) Info(msg string) {
	l.Zap.Info(msg)
}
func (l *Logger) Warn(msg string) {
	l.Zap.Warn(msg)
}
func (l *Logger) Error(msg string) {
	l.Zap.Error(msg)
}
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Zap.Debug(fmt.Sprintf(format, args...))
}
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Zap.Info(fmt.Sprintf(format, args...))
}
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Zap.Warn(fmt.Sprintf(format, args...))
}
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Zap.Error(fmt.Sprintf(format, args...))
}
func (l *Logger) Print(v ...interface{}) {
	l.Zap.Info(fmt.Sprint(v...))
}
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Zap.Info(fmt.Sprintf(format, v...))
}

// DebugRecord 分module记录日志debug级别日志
//  -- module   自定义module名称，即添加至日志JSON中的module的值，便于Es按module字段值分类别类检索
//  -- msg      日志msg
//  -- ...filed 可选的自定义添加的字段
func (l *Logger) DebugRecord(module string, msg string, filed ...zap.Field) {
	filed = append(filed, zap.String(l.recordField, module))
	l.Zap.Debug(msg, filed...)
}

// InfoRecord 分module记录日志info级别日志
//  -- module   自定义module名称，即添加至日志JSON中的module的值，便于Es按module字段值分类别类检索
//  -- msg      日志msg
//  -- ...filed 可选的自定义添加的字段
func (l *Logger) InfoRecord(module string, msg string, filed ...zap.Field) {
	filed = append(filed, zap.String(l.recordField, module))
	l.Zap.Info(msg, filed...)
}

// WarnRecord 分module记录日志warn级别日志
//  -- module   自定义module名称，即添加至日志JSON中的module的值，便于Es按module字段值分类别类检索
//  -- msg      日志msg
//  -- ...filed 可选的自定义添加的字段
func (l *Logger) WarnRecord(module string, msg string, filed ...zap.Field) {
	filed = append(filed, zap.String(l.recordField, module))
	l.Zap.Warn(msg, filed...)
}

// ErrorRecord 分module记录日志error级别日志
//  -- module   自定义module名称，即添加至日志JSON中的module的值，便于Es按module字段值分类别类检索
//  -- msg      日志msg
//  -- ...filed 可选的自定义添加的字段
func (l *Logger) ErrorRecord(module string, msg string, filed ...zap.Field) {
	filed = append(filed, zap.String(l.recordField, module))
	l.Zap.Error(msg, filed...)
}
