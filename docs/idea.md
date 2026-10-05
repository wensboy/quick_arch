# IDEA

考虑一个标准服务链路: `client.request() -> http endpoint -> muxer.route() -> middlewares -> handler -> service -> repo`.

定义:
- client.request(): 能够走一些列拦截器最终将请求发送到指定服务器
- http endpoint: 服务器本身暴露的访问端点
- muxer.route(): web framework 路由请求到实际的handler
- middlewares: handler 处理前的所有额外操作, 例如: 鉴权, 访问控制, 缓存, 事件, 加密等
- handler: 解析请求数据 -> 调用 service 获取结果 -> conn 写回 
- service: 业务逻辑计算 -> 调用 repo 获取数据 -> 计算返回
- repo: 与数据源交互控制数据的写入和取回

业务链路相关的内容大多是动态的, 其相关的依赖大多也是动态的. 例如: 
- 数据库: mysql, pgsql, sqlite, mongodb, redis, clickhouse...
- 日志: log/slog, zap, zerolog...
- 中间件: 鉴权, 访问控制, 参数校验, 缓存设置, 指标采集, 链路追踪, 日志...

考虑如下静态依赖:
- cli 启动: 跨项目一致, 统一schema和解析加载管理.
- config 管理: 跨项目一致, 统一schema和解析加载管理, 统一应用内上下文调用形式(lookup, mustLookup等).
- 上下文控制: 跨项目基本一致, 如下详细展开介绍.

## 上下文

哲学: 贯穿程序生命周期传递信息的载体, 与标准库context类似, 侧重链路的信息传递和控制. 上下文可以看作为数据视图. 例如: web framework在单个请求处理链路通常以自定义context为载体控制.

扩散到如下几个层面:
- 所有领域/模块加载 -> domain context(持久上下文)
- 信息/数据加载 -> deliver context(临时上下文)

场景示例:

程序启动时通常解析flag参数, env环境变量, config配置. 通常都遵循: flag > env > config 的覆盖体系. 假设加载后构建出一个平铺式的统一配置中心, 该配置中心本身可以是一个上下文source, 不过多考虑数据的存储形式, 只考虑通过 Lookup 和 MustLookup来获取配置的接口来与底层交互, 这就是 domain context, 也可以称为持久上下文.

程序需要在初始化阶段将template/下的模板嵌入到二进制当中, 在 root 的 .go 源代码中做嵌入后通过 main 中以 context 传递给 command 处理; 业务中间件通常需要封装并入链路形成洋葱模型, 如果没有统一形式, 在 Echo 中至少有一个二阶的 func 用于中间件的构建, 考虑该中间件受到外部配置的灵活控制, 修改将会变得复杂, 统一上下文则可以避免, 比如一个 jwt 的过期时间, 如果没有上下文, 走函数参数传递或者全局配置获取, 引入了2个复杂核心问题(参数传递不够灵活; 全局配置不够规范, 后续维护困难, 容易实现成屎山). 尝试结合: 函数参数传递上下文. 封装的业务中间件代码在获取相关的外部配置时只关注 entry 访问, 同 config 访问类似, 但又不必访问全局变量.


实现基础:
- 接口与结构的组合特性, 接口用于屏蔽差异, 规范实现, 结构用于约束上下文视图
- 解耦实现与上下文
- 所有包依赖上下文, 上下文不依赖其他包
- 依赖倒置

实现示例:

server + config + middleware

```go
package contexts
/*
    config - 配置上下文
*/
type ConfigContext interface {
    Lookup(string) (any, bool)
    MustLookup(string) any
}

/*
    middleware - 业务中间件上下文
*/
type MiddlewareContext interface {
    Set(string, any)
    Get(string) any
    Has(string) bool
    Del(string)
}

/*
    server - 服务上下文
*/
type ServerContext struct {
    ConfigContext
    MiddlewareContext
}
```

通常跨项目一致的上下文有:
- 嵌入上下文(持久, 接口): 程序启动时嵌入的数据
- 配置上下文(持久, 接口): 程序启动时加载, 运行时使用
- 指令上下文(持久, 接口): 程序启动时解析, 运行时使用
- 服务上下文(持久, 结构): 包含众多三方上下文, 运行时使用

能够约束一致的上下文有:
- 日志上下文(持久, 接口): 启动时初始化, 运行时使用
- 数据库上下文(持久, 接口): 启动时初始化, 运行时使用
- 中间件(业务集成)上下文(持久, 接口): 启动时初始化, 运行时使用
- 路由上下文(持久, 接口): 启动时初始化, 运行时使用
- 组件上下文(持久, 接口): 启动时初始化, 运行时使用

跨项目一致上下文流程:

嵌入上下文 -> 配置上下文 -> 组件上下文 -> 中间件上下文 -> 路由上下文 -> 服务上下文

注意: 日志, 数据库这类上下文本质上能够归类到组件上下文.

所以如何实际构建一个上下文?

考虑 embed 场景: 服务启动时需要针对 template/ 下的所有 .tpl 文件嵌入到最终的二进制当中, 使用嵌入上下文为后续程序使用提供.

可以定义如下上下文结构:

```go
/*
    embed context - 嵌入上下文
    1. 以当前 root 目录为基准, 通过path获取文件字节数据或者文件系统
*/

// interface 定义
type EmbedContext interface {
    PathFn(func(string) string)
    GetFile(string) ([]byte, bool)
    SetFile(string, []byte)
    GetFS(string) (*embed.FS, bool)
    SetFS(string, *embed.FS)
}

// 实现
type EmbedStore struct {
    pathFn func(string) string
    fStore map[string][]byte
    fsStore map[string]*embed.FS
}

func NewEmbedStore() *EmbedStore {
    return &EmbedStore{
        fStore: make(map[string][]byte),
        fsStore: make(map[string]*embed.FS),
    }
}

func (es *EmbedStore) PathFn(fn func(string) string) {
    es.pathFn = fn
}

func (es *EmbedStore) GetFile(path string) ([]byte, bool) {
    if es.pathFn != nil {
        path = es.pathFn(path)
    }
    v,found := es.fStore[path]
    return v, found
}

func (es *EmbedStore) GetFS(path string) (*embed.FS, bool) {
    if es.pathFn != nil {
        path = es.pathFn(path)
    }
    v,found := es.fsStore[path]
    return v, found
}

func (es *EmbedStore) SetFile(path string, content []byte) {
    if es.pathFn != nil {
        path = es.pathFn(path)
    }
    es.fStore[path] = content
}

func (es *EmbedStore) SetFS(path string, content *embed.FS) {
    if es.pathFn != nil {
        path = es.pathFn(path)
    }
    es.fsStore[path] = content
}

// 在 main.go 中
// go:embed all:template
var templateFS embed.FS

func main() {
    embedCtx := NewEmbedStore()
    embedCtx.SetFS("template", &templateFS)
    os.Exit(cmd.Execute(embedCtx))
}

// cmd.Execute(embedCtx EmbedContext) int
```

