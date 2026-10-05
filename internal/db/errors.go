package db

import errs "github.com/wensboy/quick_arch/internal/error"

var (
	ErrOpen           = errs.Register(40001, errs.CategorySystem, "数据库打开失败")
	ErrPing           = errs.Register(40002, errs.CategorySystem, "数据库连接失败")
	ErrClose          = errs.Register(40003, errs.CategorySystem, "数据库关闭失败")
	ErrDuplicate      = errs.Register(40004, errs.CategorySystem, "数据库实例重复")
	ErrNotConfigured  = errs.Register(40005, errs.CategorySystem, "数据库实例未配置")
	ErrNotSQL         = errs.Register(40006, errs.CategorySystem, "数据库实例非 SQL 类型")
	ErrUnsupported    = errs.Register(40007, errs.CategorySystem, "数据库驱动不支持")
	ErrInstanceConfig = errs.Register(40008, errs.CategorySystem, "数据库实例配置无效")

	ErrQuery    = errs.Register(40010, errs.CategorySystem, "数据库查询失败")
	ErrExec     = errs.Register(40011, errs.CategorySystem, "数据库执行失败")
	ErrTxBegin  = errs.Register(40012, errs.CategorySystem, "事务开启失败")
	ErrTxCommit = errs.Register(40013, errs.CategorySystem, "事务提交失败")
)
