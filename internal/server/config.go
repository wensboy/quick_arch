package server

import (
	"strings"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

const (
	KindREST = "rest"
	KindRPC  = "rpc"
)

const (
	KeyKind            = "server.type"
	KeyRESTAddress     = "server.rest.address"
	KeyRPCAddress      = "server.rpc.address"
	KeyRESTMiddlewares = "server.rest.middlewares"
	KeyRPCMiddlewares  = "server.rpc.middlewares"
)

const (
	DefaultKind        = KindREST
	DefaultRESTAddress = ":8080"
	DefaultRPCAddress  = ":9090"
)

// 全局中间件默认值的唯一来源, 同时供 DefaultSource 与 Load 使用.
var (
	defaultRESTMiddlewares = []string{"access_log", "recover", "request_id"}
	defaultRPCMiddlewares  = []string{"recover"}
)

// Config 是选定服务类型后的有效配置.
type Config struct {
	Kind        string
	Address     string
	Middlewares []string
}

func RegisterConfig(r *config.Registry) {
	r.Register(
		config.Entry{Key: KeyKind, Flag: "server-type", Type: config.FlagTypeString, Usage: "服务类型: rest|rpc", Env: "SERVER_TYPE", Default: DefaultKind},
		config.Entry{Key: KeyRESTAddress, Flag: "server-rest-address", Type: config.FlagTypeString, Usage: "REST 监听地址", Env: "SERVER_REST_ADDRESS", Default: DefaultRESTAddress},
		config.Entry{Key: KeyRPCAddress, Flag: "server-rpc-address", Type: config.FlagTypeString, Usage: "RPC 监听地址", Env: "SERVER_RPC_ADDRESS", Default: DefaultRPCAddress},
		config.Entry{Key: KeyRESTMiddlewares, Type: config.FlagTypeString, Usage: "REST 全局中间件, 逗号分隔", Env: "SERVER_REST_MIDDLEWARES", Default: strings.Join(defaultRESTMiddlewares, ",")},
		config.Entry{Key: KeyRPCMiddlewares, Type: config.FlagTypeString, Usage: "RPC 全局中间件, 逗号分隔", Env: "SERVER_RPC_MIDDLEWARES", Default: strings.Join(defaultRPCMiddlewares, ",")},
	)
}

// Load 依据 server.type 选取对应服务类型的地址与中间件配置.
func Load(cfg context2.ConfigContext) Config {
	kind := strings.ToLower(config.String(cfg, KeyKind, DefaultKind))

	switch kind {
	case KindRPC:
		return Config{
			Kind:        KindRPC,
			Address:     config.String(cfg, KeyRPCAddress, DefaultRPCAddress),
			Middlewares: config.List(cfg, KeyRPCMiddlewares, defaultRPCMiddlewares),
		}
	default:
		return Config{
			Kind:        kind,
			Address:     config.String(cfg, KeyRESTAddress, DefaultRESTAddress),
			Middlewares: config.List(cfg, KeyRESTMiddlewares, defaultRESTMiddlewares),
		}
	}
}
