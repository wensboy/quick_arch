package server

import errs "github.com/wensboy/quick_arch/internal/error"

var (
	ErrUnsupportedKind = errs.Register(50001, errs.CategorySystem, "服务类型不支持")
	ErrAppContext      = errs.Register(50002, errs.CategorySystem, "应用上下文缺失")
	ErrListen          = errs.Register(50003, errs.CategorySystem, "服务监听失败")
	ErrTLS             = errs.Register(50004, errs.CategorySystem, "服务 TLS 配置无效")
)
