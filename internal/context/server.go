package context

type ServerContext struct {
	DatabaseContext
	LogContext
	ConfigContext
	MiddlewareContext
	EmbedContext
}

func NewServerContext(cfg ConfigContext, log LogContext, db DatabaseContext, middleware MiddlewareContext, embeds EmbedContext) *ServerContext {
	return &ServerContext{
		ConfigContext:     cfg,
		LogContext:        log,
		DatabaseContext:   db,
		MiddlewareContext: middleware,
		EmbedContext:      embeds,
	}
}
