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
	KeyRESTMiddlewares = "server.rest.middlewares"
	KeyRPCAddress      = "server.rpc.address"
	// rpc 侧一元与流分别配置, 中间件名称带 _rpc 后缀.
	KeyRPCUnaryMiddlewares  = "server.rpc.unary_middlewares"
	KeyRPCStreamMiddlewares = "server.rpc.stream_middlewares"
	// rpc 反射与传输层 TLS.
	KeyRPCReflection  = "server.rpc.reflection"
	KeyRPCTLSEnabled  = "server.rpc.tls.enabled"
	KeyRPCTLSCertFile = "server.rpc.tls.cert_file"
	KeyRPCTLSKeyFile  = "server.rpc.tls.key_file"
)

const (
	DefaultKind          = KindREST
	DefaultRESTAddress   = ":8080"
	DefaultRPCAddress    = ":9090"
	DefaultRPCReflection = true
)

// 全局中间件默认值的唯一来源, 同时供 DefaultSource 与 Load 使用.
var (
	defaultRESTMiddlewares      = []string{"access_log", "recover", "request_id"}
	defaultRPCUnaryMiddlewares  = []string{"recover_rpc"}
	defaultRPCStreamMiddlewares = []string{"recover_rpc"}
)

// Config 是选定服务类型后的有效配置.
type Config struct {
	Kind        string
	Address     string
	Middlewares []string // rest 生效
	// rpc 一元 / 流两种形态各自生效.
	UnaryMiddlewares  []string
	StreamMiddlewares []string
	// rpc 专用: 服务反射与传输层 TLS.
	Reflection  bool
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string
}

func RegisterConfig(r *config.Registry) {
	r.Register(
		config.Entry{Key: KeyKind, Flag: "server-type", Type: config.FlagTypeString, Usage: "服务类型: rest|rpc", Env: "SERVER_TYPE", Default: DefaultKind},
		config.Entry{Key: KeyRESTAddress, Flag: "server-rest-address", Type: config.FlagTypeString, Usage: "REST 监听地址", Env: "SERVER_REST_ADDRESS", Default: DefaultRESTAddress},
		config.Entry{Key: KeyRPCAddress, Flag: "server-rpc-address", Type: config.FlagTypeString, Usage: "RPC 监听地址", Env: "SERVER_RPC_ADDRESS", Default: DefaultRPCAddress},
		config.Entry{Key: KeyRESTMiddlewares, Type: config.FlagTypeString, Usage: "REST 全局中间件, 逗号分隔", Env: "SERVER_REST_MIDDLEWARES", Default: strings.Join(defaultRESTMiddlewares, ",")},
		config.Entry{Key: KeyRPCUnaryMiddlewares, Type: config.FlagTypeString, Usage: "RPC 一元拦截器, 逗号分隔 (名称带 _rpc 后缀)", Env: "SERVER_RPC_UNARY_MIDDLEWARES", Default: strings.Join(defaultRPCUnaryMiddlewares, ",")},
		config.Entry{Key: KeyRPCStreamMiddlewares, Type: config.FlagTypeString, Usage: "RPC 流拦截器, 逗号分隔 (名称带 _rpc 后缀)", Env: "SERVER_RPC_STREAM_MIDDLEWARES", Default: strings.Join(defaultRPCStreamMiddlewares, ",")},
		config.Entry{Key: KeyRPCReflection, Type: config.FlagTypeBool, Usage: "RPC 开启服务反射 (供 grpcurl 免 proto 调试)", Env: "SERVER_RPC_REFLECTION", Default: DefaultRPCReflection},
		config.Entry{Key: KeyRPCTLSEnabled, Type: config.FlagTypeBool, Usage: "RPC 开启传输层 TLS", Env: "SERVER_RPC_TLS_ENABLED"},
		config.Entry{Key: KeyRPCTLSCertFile, Type: config.FlagTypeString, Usage: "RPC TLS 证书文件", Env: "SERVER_RPC_TLS_CERT_FILE"},
		config.Entry{Key: KeyRPCTLSKeyFile, Type: config.FlagTypeString, Usage: "RPC TLS 私钥文件", Env: "SERVER_RPC_TLS_KEY_FILE"},
	)
}

// Load 依据 server.type 选取对应服务类型的地址与中间件配置.
func Load(cfg context2.ConfigContext) Config {
	kind := strings.ToLower(config.String(cfg, KeyKind, DefaultKind))

	switch kind {
	case KindRPC:
		return Config{
			Kind:              KindRPC,
			Address:           config.String(cfg, KeyRPCAddress, DefaultRPCAddress),
			UnaryMiddlewares:  config.List(cfg, KeyRPCUnaryMiddlewares, defaultRPCUnaryMiddlewares),
			StreamMiddlewares: config.List(cfg, KeyRPCStreamMiddlewares, defaultRPCStreamMiddlewares),
			Reflection:        config.Bool(cfg, KeyRPCReflection, DefaultRPCReflection),
			TLSEnabled:        config.Bool(cfg, KeyRPCTLSEnabled, false),
			TLSCertFile:       config.String(cfg, KeyRPCTLSCertFile, ""),
			TLSKeyFile:        config.String(cfg, KeyRPCTLSKeyFile, ""),
		}
	default:
		return Config{
			Kind:        kind,
			Address:     config.String(cfg, KeyRESTAddress, DefaultRESTAddress),
			Middlewares: config.List(cfg, KeyRESTMiddlewares, defaultRESTMiddlewares),
		}
	}
}
