package middleware

import (
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

func NewFilter(include, exclude []string) Filter {
	return Filter{include: include, exclude: exclude}
}

func (f Filter) Allow(path string) bool {
	for _, prefix := range f.exclude {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	if len(f.include) == 0 {
		return true
	}
	for _, prefix := range f.include {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Wrap 让中间件只在 Filter 命中的路径上执行, 未命中时直接放行.
func (f Filter) Wrap(mw echo.MiddlewareFunc) echo.MiddlewareFunc {
	if len(f.include) == 0 && len(f.exclude) == 0 {
		return mw
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		handled := mw(next)
		return func(c *echo.Context) error {
			if !f.Allow(c.Request().URL.Path) {
				return next(c)
			}
			return handled(c)
		}
	}
}

// Filter 控制中间件在哪些路由上生效: exclude 优先于 include, include 为空表示默认全部生效.
// 匹配对象由调用方语义决定: rest 传 URI 路径, rpc 传函数名.
type Filter struct {
	include []string
	exclude []string
}

func filterOf(cfg context2.ConfigContext, name string) Filter {
	return NewFilter(
		config.List(cfg, keyInclude(name), nil),
		config.List(cfg, keyExclude(name), nil),
	)
}
