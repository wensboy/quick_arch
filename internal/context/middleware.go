package context

import (
	"github.com/labstack/echo/v5"
	"google.golang.org/grpc"
)

// MiddlewareFactory 依据中间件上下文构建中间件, 执行后即可得到可复用的 echo.MiddlewareFunc.
type MiddlewareFactory func(MiddlewareContext) []echo.MiddlewareFunc

// UnaryInterceptorFactory 是中间件的 gRPC 一元拦截器形态.
type UnaryInterceptorFactory func(MiddlewareContext) []grpc.UnaryServerInterceptor

// StreamInterceptorFactory 是中间件的 gRPC 流拦截器形态.
type StreamInterceptorFactory func(MiddlewareContext) []grpc.StreamServerInterceptor

// 中间件值区中预置的第三方模块上下文.
const (
	MiddlewareValueLog    = "ctx.log"
	MiddlewareValueConfig = "ctx.config"
	MiddlewareValueDB     = "ctx.db"
)

// MiddlewareContext 统一管理业务中间件所需的"值"与"函数":
//   - 值: 配置项等, 用于改变中间件行为
//   - 函数: 延迟构建并按名复用, 使 router/server 无需依赖具体中间件实现
//
// 同一中间件按传输与调用形态分别注册, 各自独立解析:
//   - rest  : echo.MiddlewareFunc
//   - rpc一元: grpc.UnaryServerInterceptor
//   - rpc流  : grpc.StreamServerInterceptor
//
// 命名约定: rpc 侧名称在 rest 名称基础上加 "_rpc" 后缀用于划分.
// 过滤规则对 rest 是 URI 路径, 对 rpc 是函数名.
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

	RegisterUnary(name string, factory UnaryInterceptorFactory)
	ResolveUnary(names ...string) []grpc.UnaryServerInterceptor

	RegisterStream(name string, factory StreamInterceptorFactory)
	ResolveStream(names ...string) []grpc.StreamServerInterceptor
}
