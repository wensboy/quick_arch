package middleware

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

func recoverFactory() context2.MiddlewareFactory {
	return func(mc context2.MiddlewareContext) []echo.MiddlewareFunc {
		return []echo.MiddlewareFunc{func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) (err error) {
				defer func() {
					if r := recover(); r != nil {
						logAt(mc, "error", "panic recovered", "path", c.Request().URL.Path, "panic", r)
						err = echo.NewHTTPError(http.StatusInternalServerError, "internal error")
					}
				}()
				return next(c)
			}
		}}
	}
}

func requestIDFactory() context2.MiddlewareFactory {
	var seq atomic.Uint64
	return func(context2.MiddlewareContext) []echo.MiddlewareFunc {
		return []echo.MiddlewareFunc{func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				c.Response().Header().Set("X-Request-Id", strconv.FormatUint(seq.Add(1), 10))
				return next(c)
			}
		}}
	}
}

func accessLogFactory(cfg context2.ConfigContext) context2.MiddlewareFactory {
	return func(mc context2.MiddlewareContext) []echo.MiddlewareFunc {
		filter := filterOf(cfg, NameAccessLog)
		return []echo.MiddlewareFunc{filter.Wrap(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				start := time.Now()
				err := next(c)
				// 错误响应由 echo 在中间件链路之后写出, 故按 echo 规则推导状态码.
				_, status := echo.ResolveResponseStatus(c.Response(), err)
				logAt(mc, "info", "access",
					"method", c.Request().Method,
					"path", c.Request().URL.Path,
					"status", status,
					"cost", time.Since(start).String(),
				)
				return err
			}
		})}
	}
}

// debugFactory 用于验证中间件是否被正确集成: 命中路径时回写标记头.
func debugFactory(cfg context2.ConfigContext) context2.MiddlewareFactory {
	return func(context2.MiddlewareContext) []echo.MiddlewareFunc {
		filter := filterOf(cfg, NameDebug)
		return []echo.MiddlewareFunc{filter.Wrap(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				c.Response().Header().Add("X-Debug-Middleware", NameDebug)
				return next(c)
			}
		})}
	}
}

func logAt(mc context2.MiddlewareContext, level, msg string, kv ...any) {
	if mc == nil {
		return
	}
	logger, ok := mc.Get(context2.MiddlewareValueLog).(context2.LogContext)
	if !ok || logger == nil {
		return
	}
	logger.Log(level, msg, kv...)
}
