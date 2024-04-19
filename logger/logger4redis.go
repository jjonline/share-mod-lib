package logger

import (
	"bytes"
	"context"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"sync"
)

// RedisHook redis hook implement
type RedisHook struct{}

// BeforeProcess redis执行命令前hook
func (r RedisHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return ctx, nil
}

// AfterProcess redis执行命令后hook
func (r RedisHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	if zapLogger != nil {
		if err := cmd.Err(); err != nil && err != redis.Nil {
			zapLogger.Error("redis command error",
				zap.String("command", cmd.String()),
				zap.String("module", "redis"))
		} else {
			zapLogger.Debug("redis command success",
				zap.String("command", cmd.String()),
				zap.String("module", "redis"))
		}
	}
	return nil
}

// BeforeProcessPipeline redis执行pipe前hook
func (r RedisHook) BeforeProcessPipeline(ctx context.Context, cmdItems []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

// AfterProcessPipeline redis执行pipe后hook
func (r RedisHook) AfterProcessPipeline(ctx context.Context, cmdItems []redis.Cmder) error {
	if zapLogger != nil {
		var err error
		buf := getBuffer()
		defer putBuffer(buf)
		var isError bool
		for i := range cmdItems {
			buf.WriteString(cmdItems[i].String())
			if err = cmdItems[i].Err(); err != nil && err != redis.Nil {
				isError = true
			}
			buf.WriteByte('|')
		}
		if isError {
			zapLogger.Error("redis pipeline error",
				zap.String("command", buf.String()),
				zap.String("module", "redis"))
		} else {
			zapLogger.Debug("redis pipeline success",
				zap.String("command", buf.String()),
				zap.String("module", "redis"))
		}
	}
	return nil
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// getBuffer 获取一个 buffer
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer 释放一个buffer
func putBuffer(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bufferPool.Put(buf)
	}
}
