package context

import "context"

type AppContext struct {
	context.Context
	EmbedContext
	ConfigContext
	LogContext
}

type appKey struct{}

// Value 使 AppContext 能在上下文链中被取出, 其余键委托给内嵌 Context.
func (a AppContext) Value(key any) any {
	if _, ok := key.(appKey); ok {
		return a
	}
	if a.Context == nil {
		return nil
	}
	return a.Context.Value(key)
}

func AppFrom(ctx context.Context) (AppContext, bool) {
	if ctx == nil {
		return AppContext{}, false
	}
	app, ok := ctx.Value(appKey{}).(AppContext)
	return app, ok
}
