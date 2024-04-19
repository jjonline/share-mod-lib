package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net"
	"os"
	"time"
)

// ++++++++++
// 初始化zap
// ++++++++++

// zapLogger init初始化后的内部zapLogger全局句柄
var zapLogger *zap.Logger

// newZap 初始化一个zapLogger
func newZap(logLevel, logPath string) *zap.Logger {
	level := zap.DebugLevel
	switch logLevel {
	case "debug", "trace":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "dpanic":
		level = zap.DPanicLevel
	case "panic":
		level = zap.PanicLevel
	case "fatal":
		level = zap.FatalLevel
	}

	// 获取log输出目录
	var logWriter string
	if logPath == "stderr" {
		logWriter = "stderr"
	} else if logPath == "stdout" {
		logWriter = "stdout"
	} else {
		if !pathExists(logPath) {
			err := os.MkdirAll(logPath, os.ModePerm)
			if err != nil {
				panic(fmt.Sprintf("logPath: %s MkdirAll error: %s", logPath, err.Error()))
			}
		}
		logWriter = logPath + "/" + time.Now().Format("2006-01-02") + ".log"
	}

	// zap Microseconds decoder
	var microDurationEncoder = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendInt64(d.Nanoseconds() / 1e3)
	}

	c := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "json", // console 控制台打印风格  json 输出json字符串
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "msg",
			LevelKey:   "level",
			TimeKey:    "ts",
			//NameKey:    "logger",
			//CallerKey:      "caller",
			//StacktraceKey:  "trace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: microDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
		},
		OutputPaths:      []string{logWriter},
		ErrorOutputPaths: []string{logWriter},
		InitialFields:    nil,
	}

	var err error
	zapLogger, err = c.Build()
	if err != nil {
		panic(err.Error())
	}

	// set basic host filed
	zapLogger = zapLogger.With(zap.String("server", hostIp()))
	return zapLogger
}

// hostIp 本机IP
func hostIp() string {
	if adds, err := net.InterfaceAddrs(); err == nil {
		for i := range adds {
			if ipNet, ok := adds[i].(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil {
					return ipNet.IP.String()
				}
			}
		}
	}
	return "0.0.0.0"
}

//判断文件夹是否存在
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
