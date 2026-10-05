package config

import errs "github.com/wensboy/quick_arch/internal/error"

var (
	ErrReadFile   = errs.Register(31001, errs.CategorySystem, "配置文件读取失败")
	ErrParseFile  = errs.Register(31002, errs.CategorySystem, "配置文件解析失败")
	ErrMissingKey = errs.Register(31003, errs.CategorySystem, "配置项缺失")
)
