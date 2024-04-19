package robot

import (
	"context"
	"time"
)

var (
	UTCZone8         = "Asia/Hong_Kong"
	UTCZone8Location = time.FixedZone(UTCZone8, 8*3600)
)

type Robot interface {
	Info(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error)
	Warning(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error)
	Error(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error)
	Message(ctx context.Context, bg, title, markdownText string, t time.Time, links ...LinkItem) (err error)
	Once(webhook, secret string) Robot
}
