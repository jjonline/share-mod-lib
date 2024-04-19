# Logger 组件

## 说明

本包为zap日志组件封装，用于golang项目统一日志输出和收集。

## 使用示例

* 日志组件使用：`zap`
* 日志输出选项：`stdout` `stderr` | 日志目录 (指定存储目录)
* 日志级别选项：`panic` `fatal` `error` `warning` `info` `debug`
* 生产环境级别：`info`
* 开发测试级别：`debug`
* 日志输出格式：`json`

````
// instance
// 第一参数为日志级别
// 第二参数为日志存储路径，文件形式记录日志则是文件路径，标准输出则 stderr
var log = logger.New("debug", "stderr")

// 直接使用记录一段字符串日志
log.Debug("debug data")
log.Error("debug error")

// 直接使用底层zap记录多字段日志，性能更佳，推荐方式
log.Zap.Info(
    "msg",
    zap.String("module", "module-can-use-kibana"),
    zap.String("sql", "your sql"),
    zap.Int("num", 101),
)
````

> 本库亦封装实现了`redis`、`elastic`、`gorm`的logger实现，以使用底层`zap`日志组件。
