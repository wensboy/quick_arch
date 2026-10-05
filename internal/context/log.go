package context

// LogContext 统一日志入口.
// 级别通过 level 参数显式传入, 不暴露 Info/Warn 等与级别命名绑定的方法.
type LogContext interface {
	Log(level string, msg string, kv ...any)
	Logf(level string, format string, args ...any)
	Sync() error
}
