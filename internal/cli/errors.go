package cli

import errs "github.com/wensboy/quick_arch/internal/error"

var (
	ErrParseSource = errs.Register(30001, errs.CategorySystem, "命令源解析失败")
	ErrReadSource  = errs.Register(30002, errs.CategorySystem, "命令源读取失败")
)
