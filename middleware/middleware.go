package middleware

import context2 "github.com/wensboy/quick_arch/internal/context"

const (
	NameRecover   = "recover"
	NameRequestID = "request_id"
	NameAccessLog = "access_log"
	NameDebug     = "debug"
)

// RegisterMiddlewares 将全部内置中间件注册到中间件上下文 (延迟构建, 首次 Resolve 时生效).
func RegisterMiddlewares(mc context2.MiddlewareContext, cfg context2.ConfigContext) {
	mc.Register(NameRecover, recoverFactory())
	mc.Register(NameRequestID, requestIDFactory())
	mc.Register(NameAccessLog, accessLogFactory(cfg))
	mc.Register(NameDebug, debugFactory(cfg))
}
