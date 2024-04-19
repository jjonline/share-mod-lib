package logger

import (
	"go.uber.org/zap"
	"time"
)

// DBLogger gorm日志记录器
type DBLogger struct{}

// Print db log record
func (l DBLogger) Print(values ...interface{}) {
	if zapLogger == nil {
		return
	}

	if len(values) < 2 {
		return
	}

	switch values[0] {
	case "sql":
		zapLogger.Info(
			"gorm.debug.sql",
			zap.String("module", "sql"),
			zap.String("query", values[3].(string)),
			zap.Any("values", values[4]),
			zap.Duration("duration", values[2].(time.Duration)),
			zap.Int64("rows", values[5].(int64)),
			zap.String("source", values[1].(string)), // if AddCallerSkip(6) is well defined, we can safely remove this field
		)
	default:
		zapLogger.Info(
			"gorm.debug.other",
			zap.String("module", "sql"),
			zap.Any("values", values[2:]),
			zap.String("source", values[1].(string)), // if AddCallerSkip(6) is well defined, we can safely remove this field
		)
	}
}
