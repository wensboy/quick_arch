package server

import errs "github.com/wensboy/quick_arch/internal/error"

// New 依据配置构建服务实例; routers 仅在 rest 类型下挂载, services 仅在 rpc 类型下挂载.
func New(cfg Config, routers []Router, services ...Service) (Server, error) {
	switch cfg.Kind {
	case KindREST:
		return NewRestServer().
			WithAddress(cfg.Address).
			Use(cfg.Middlewares...).
			Mount(routers...), nil
	case KindRPC:
		return NewRpcServer().
			WithAddress(cfg.Address).
			WithReflection(cfg.Reflection).
			WithTLS(cfg.TLSEnabled, cfg.TLSCertFile, cfg.TLSKeyFile).
			UseUnary(cfg.UnaryMiddlewares...).
			UseStream(cfg.StreamMiddlewares...).
			Mount(services...), nil
	default:
		return nil, errs.New(ErrUnsupportedKind).With("kind", cfg.Kind)
	}
}
