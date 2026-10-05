package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
)

type stubRouter struct {
	prefix string
	path   string
	trace  *[]string
}

func (r stubRouter) Prefix() string { return r.prefix }

func (r stubRouter) Mount(g *echo.Group, ctx context2.ServerContext) {
	g.GET(r.path, func(c *echo.Context) error {
		if r.trace != nil {
			*r.trace = append(*r.trace, "handler")
		}
		return c.String(http.StatusOK, "suffix="+ctx.Get("suffix").(string))
	})
}

func TestRestServer_Setup(t *testing.T) {
	var trace []string

	store := NewMiddlewareStore()
	store.Set("suffix", "v1")
	store.Register("mark", func(context2.MiddlewareContext) []echo.MiddlewareFunc {
		return []echo.MiddlewareFunc{func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				trace = append(trace, "middleware")
				return next(c)
			}
		}}
	})

	ctx := context2.NewServerContext(nil, nil, nil, store, nil)
	srv := NewRestServer().Use("mark").Mount(stubRouter{prefix: "/v1", path: "/ping", trace: &trace})
	srv.Setup(*ctx)

	rec := httptest.NewRecorder()
	srv.muxer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/ping", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "suffix=v1" {
		t.Fatalf("resp = %d %q", rec.Code, rec.Body.String())
	}
	if len(trace) != 2 || trace[0] != "middleware" || trace[1] != "handler" {
		t.Fatalf("trace = %v, want [middleware handler]", trace)
	}
}

func TestRestServer_Setup_IgnoresUnknownMiddleware(t *testing.T) {
	store := NewMiddlewareStore()
	store.Set("suffix", "v1")
	ctx := context2.NewServerContext(nil, nil, nil, store, nil)

	srv := NewRestServer().Use("not-registered").Mount(stubRouter{prefix: "", path: "/ping"})
	srv.Setup(*ctx)

	rec := httptest.NewRecorder()
	srv.muxer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRestServer_Setup_NoMiddlewareContext(t *testing.T) {
	srv := NewRestServer().Use("mark")
	srv.Setup(context2.ServerContext{})
}

func TestRestServer_ErrorHandler(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"not found", errs.New(errs.ErrNotFound), http.StatusNotFound, `{"code":20001,"message":"资源不存在"}`},
		{"permission", errs.New(errs.ErrPermission), http.StatusForbidden, `{"code":20003,"message":"无权限"}`},
		{"internal", errs.New(errs.ErrInternal), http.StatusInternalServerError, `{"code":-10000,"message":"内部错误"}`},
		{"echo route miss", echo.ErrNotFound, http.StatusNotFound, `{"code":20001,"message":"资源不存在"}`},
		{"echo unmapped status", echo.ErrMethodNotAllowed, http.StatusMethodNotAllowed, `{"code":-10000,"message":"内部错误"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewRestServer().Mount(errRouter{err: tc.err})
			srv.Setup(context2.ServerContext{})

			rec := httptest.NewRecorder()
			srv.muxer.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := strings.TrimSpace(rec.Body.String()); got != tc.wantBody {
				t.Fatalf("body = %s, want %s", got, tc.wantBody)
			}
		})
	}
}

type errRouter struct{ err error }

func (errRouter) Prefix() string { return "" }

func (r errRouter) Mount(g *echo.Group, ctx context2.ServerContext) {
	g.GET("/boom", func(*echo.Context) error { return r.err })
}
