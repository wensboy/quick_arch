# server 相关

当前 server 启动形式如下: `third modules context -> server context -> server -> runner`

- third modules context: database, log, config, middleware, router...
- server context: 核心用于 setup 一个 server 配置加载
- server: 窄接口抽象, 包含 server 的完整的生命周期
- runner: 通过 server 接口控制启动, 信号监听, 优雅关闭

假设当前 database, log, config 都实现(当前确实实现), 考虑: middleware, router.

## middleware

Echo 中使用业务中间件一般封装形式: `func <middleware_name>(<middleware_context>) echo.MiddlewareFunc`

聚焦考虑: `middleware_context` 部分, 关注业务中间件需要:
- 必要的值: 配置改变中间件行为
- 必要的处理函数: 延迟计算, 中间件复用和统一管理

因此, 中间件上下文本质上处理对一个中间件需要的值和函数的管理和传递. 当上述的函数执行后立刻得到一个函数指针, 其指向一个合法的echo中间件函数, 可以用于多路由直接复用. 可以解耦router必须依赖middleware, 以达到统一地方注册中间件的效果.

## router

简单设计可以考虑在 `router` 目录下通过 prefix 来作为一个版本, 内部包含一个 `mountRouter(*echo.Group, ServerContext)`, 在内部注册所有的路由, 因此将所有的服务的 handler 引入限制在 router 层面. 当然也可以考虑通过高级的上下文统一形式来处理一些拓展, 比如在启动时将所有的注册情况打印到console, 但是显然可以通过注册scalar或者swagger路由来实现.

