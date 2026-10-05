package context

import "github.com/labstack/echo/v5"

// MiddlewareFactory 依据中间件上下文构建中间件, 执行后即可得到可复用的 echo.MiddlewareFunc.
type MiddlewareFactory func(MiddlewareContext) []echo.MiddlewareFunc

// 中间件值区中预置的第三方模块上下文.
const (
	MiddlewareValueLog    = "ctx.log"
	MiddlewareValueConfig = "ctx.config"
	MiddlewareValueDB     = "ctx.db"
)

// MiddlewareContext 统一管理业务中间件所需的"值"与"函数":
//   - 值: 配置项等, 用于改变中间件行为
//   - 函数: 延迟构建并按名复用, 使 router 无需依赖具体中间件实现
//
// 方法名与 ConfigContext 保持不重叠, 避免 ServerContext 组合时产生歧义.
type MiddlewareContext interface {
	Set(key string, value any)
	Get(key string) any
	Has(key string) bool
	Del(key string)

	Register(name string, factory MiddlewareFactory)
	Registered() []string
	Resolve(names ...string) []echo.MiddlewareFunc
}
