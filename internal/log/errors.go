package log

import errs "github.com/wensboy/quick_arch/internal/error"

var (
	ErrInvalidLevel  = errs.Register(32001, errs.CategorySystem, "日志级别非法")
	ErrInvalidOutput = errs.Register(32002, errs.CategorySystem, "日志输出目标非法")
)
