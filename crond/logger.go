package crond

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// cronLog cron日志记录器
type cronLog struct {
	logger *zap.Logger
}

// Info 信息级别日志输出
func (l cronLog) Info(msg string, keysAndValues ...interface{}) {
	// record
	var fields []zapcore.Field
	fields = append(fields, zap.String("module", module))
	for key, val := range keysAndValues {
		if "now" == val {
			fields = append(fields, zap.Any("execute_time", keysAndValues[key+1]))
		}
		if "entry" == val {
			entryID := keysAndValues[key+1]
			entryIntID := int(entryID.(cron.EntryID))
			fields = append(fields, zap.Int("entry_id", entryIntID))

			// 取当前command的签名添加日志字段
			if command, exist := registeredCommand[entryIntID]; exist {
				fields = append(fields, zap.String("signature", command.Signature()))
			}
		}
		if "next" == val {
			fields = append(fields, zap.Any("next_time", keysAndValues[key+1]))
		}
	}

	// 忽略掉wake类型
	if msg != "wake" {
		l.logger.Info("crontab.log."+msg, fields...)
	}
}

// Error 错误级别日志输出
func (l cronLog) Error(err error, msg string, keysAndValues ...interface{}) {
	var fields []zapcore.Field
	fields = append(fields, zap.String("module", module), zap.Error(err))
	for key, val := range keysAndValues {
		if "stack" == val {
			fields = append(fields, zap.Any("stack", keysAndValues[key+1]))
		}
	}
	l.logger.Error("crontab.error."+msg, fields...)
}
