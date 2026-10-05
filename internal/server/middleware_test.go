package server

import (
	"testing"

	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

func passThrough() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
}

func TestMiddlewareStore_Values(t *testing.T) {
	store := NewMiddlewareStore()

	if store.Has("ttl") {
		t.Fatal("Has should miss unset key")
	}
	if store.Get("nope") != nil {
		t.Fatal("Get should return nil for missing key")
	}

	store.Set("ttl", 30)
	if !store.Has("ttl") || store.Get("ttl") != 30 {
		t.Fatalf("ttl = %v", store.Get("ttl"))
	}

	store.Del("ttl")
	if store.Has("ttl") {
		t.Fatal("Del should remove key")
	}
}

func TestMiddlewareStore_Resolve_LazyAndCached(t *testing.T) {
	store := NewMiddlewareStore()
	store.Set("ttl", 30)

	builds := 0
	store.Register("noop", func(mc context2.MiddlewareContext) []echo.MiddlewareFunc {
		builds++
		if mc.Get("ttl") != 30 {
			t.Errorf("factory should read values from context, got %v", mc.Get("ttl"))
		}
		return []echo.MiddlewareFunc{passThrough()}
	})

	if builds != 0 {
		t.Fatalf("factory should be lazy, builds=%d", builds)
	}
	if got := store.Resolve("noop"); len(got) != 1 || builds != 1 {
		t.Fatalf("resolve = %d funcs, builds=%d", len(got), builds)
	}

	_ = store.Resolve("noop")
	if builds != 1 {
		t.Fatalf("resolved result should be cached, builds=%d", builds)
	}

	store.Register("noop", func(context2.MiddlewareContext) []echo.MiddlewareFunc {
		builds++
		return []echo.MiddlewareFunc{passThrough()}
	})
	_ = store.Resolve("noop")
	if builds != 2 {
		t.Fatalf("re-register should invalidate cache, builds=%d", builds)
	}

	store.Reset()
	_ = store.Resolve("noop")
	if builds != 3 {
		t.Fatalf("Reset should drop cache, builds=%d", builds)
	}
}

func TestMiddlewareStore_Resolve_SkipsUnknown(t *testing.T) {
	store := NewMiddlewareStore()
	if got := store.Resolve("missing"); len(got) != 0 {
		t.Fatalf("Resolve(missing) = %v, want empty", got)
	}
}

func TestMiddlewareStore_Resolve_Order(t *testing.T) {
	store := NewMiddlewareStore()
	var order []string

	register := func(name string) {
		store.Register(name, func(context2.MiddlewareContext) []echo.MiddlewareFunc {
			return []echo.MiddlewareFunc{func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error {
					order = append(order, name)
					return next(c)
				}
			}}
		})
	}
	register("a")
	register("b")

	chain := store.Resolve("b", "unknown", "a")
	if len(chain) != 2 {
		t.Fatalf("chain = %d, want 2", len(chain))
	}
	handler := chain[0](chain[1](func(*echo.Context) error { return nil }))
	_ = handler(nil)
	if len(order) != 2 || order[0] != "b" || order[1] != "a" {
		t.Fatalf("order = %v, want [b a]", order)
	}
}

func TestMiddlewareStore_Registered(t *testing.T) {
	store := NewMiddlewareStore()
	store.Register("b", func(context2.MiddlewareContext) []echo.MiddlewareFunc { return nil })
	store.Register("a", func(context2.MiddlewareContext) []echo.MiddlewareFunc { return nil })

	got := store.Registered()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("Registered = %v, want [a b]", got)
	}
}
