package logger

import (
	"fmt"
	"go.uber.org/zap"
	"net/url"
)

// EsLogger es 日志记录器
type EsLogger struct{}

// Printf Print es log record
func (l EsLogger) Printf(format string, values ...interface{}) {
	if zapLogger == nil {
		return
	}
	switch len(values) {
	case 1:
		// cluster mode
		//elastic: %s joined the cluster
		zapLogger.Info(
			"es.other",
			zap.String("module", "elastic"),
			zap.String("tips", fmt.Sprintf(format, values[0])),
		)
	case 4:
		//"%s %s [status:%d, request:%.3fs]"
		zapLogger.Info(
			"es.request",
			zap.String("module", "elastic"),
			zap.String("method", values[0].(string)),
			zap.String("url", values[1].(*url.URL).String()),
			zap.Int("http_status", values[2].(int)),
			zap.Float64("duration", (values[3]).(float64)*1000000),
		)
	default:
		// cluster mode
		//elastic: client started
		//elastic: client stopped
		//other
		zapLogger.Info(
			"es.other",
			zap.String("module", "elastic"),
			zap.String("tips", format),
		)
	}
}
