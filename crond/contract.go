package crond

// CronTask 定时任务类契约
type CronTask interface {
	Signature() string // Signature 定时任务名称，即赋予定时任务的一个名称便于日志里识别
	Rule() string      // CronRule  定时规则：`Second | Minute | Hour | Dom (day of month) | Month | Dow (day of week)`
	Execute() error    // Execute   执行入口，返回nil执行成功，返回error或发生panic执行失败
}
