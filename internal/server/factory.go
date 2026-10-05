package server

import errs "github.com/wensboy/quick_arch/internal/error"

// New 依据配置构建服务实例; routers 仅在 rest 类型下挂载.
func New(cfg Config, routers []Router) (Server, error) {
	switch cfg.Kind {
	case KindREST:
		return NewRestServer().
			WithAddress(cfg.Address).
			Use(cfg.Middlewares...).
			Mount(routers...), nil
	default:
		return nil, errs.New(ErrUnsupportedKind).With("kind", cfg.Kind)
	}
}
