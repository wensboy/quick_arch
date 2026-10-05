package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

func TestAccessLogFactory_LogsResponseStatus(t *testing.T) {
	teapot := echo.NewHTTPError(http.StatusTeapot, "teapot")
	boom := errors.New("boom")

	cases := []struct {
		name       string
		inner      func(*echo.Context) error
		wantStatus int
		wantErr    error
	}{
		{"success", func(*echo.Context) error { return nil }, http.StatusOK, nil},
		{
			"committed",
			func(c *echo.Context) error { c.Response().WriteHeader(http.StatusNotFound); return nil },
			http.StatusNotFound,
			nil,
		},
		{"http error", func(*echo.Context) error { return teapot }, http.StatusTeapot, teapot},
		{"plain error", func(*echo.Context) error { return boom }, http.StatusInternalServerError, boom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger := &recordingLogger{}
			middlewares := accessLogFactory(nil)(stubMiddlewareContext{log: logger})
			if len(middlewares) != 1 {
				t.Fatalf("middlewares = %d, want 1", len(middlewares))
			}

			e := echo.New()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil), httptest.NewRecorder())
			err := middlewares[0](tc.inner)(c)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if logger.msg != "access" {
				t.Fatalf("msg = %q, want access", logger.msg)
			}
			if got := kvValue(logger.kv, "status"); got != tc.wantStatus {
				t.Fatalf("status = %v, want %d", got, tc.wantStatus)
			}
			if got := kvValue(logger.kv, "path"); got != "/api/v1/ping" {
				t.Fatalf("path = %v, want /api/v1/ping", got)
			}
		})
	}
}

// TestAccessLogFactory_LogsRecoveredPanic 校验默认顺序 (access_log 外层, recover 内层) 下 panic 请求也会被记录.
func TestAccessLogFactory_LogsRecoveredPanic(t *testing.T) {
	logger := &recordingLogger{}
	mc := stubMiddlewareContext{log: logger}

	accessLog := accessLogFactory(nil)(mc)[0]
	recoverMiddleware := recoverFactory()(mc)[0]

	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil), httptest.NewRecorder())
	err := accessLog(recoverMiddleware(func(*echo.Context) error { panic("boom") }))(c)

	if err == nil {
		t.Fatal("err = nil, want panic converted to error")
	}
	if logger.msg != "access" {
		t.Fatalf("msg = %q, want access", logger.msg)
	}
	if got := kvValue(logger.kv, "status"); got != http.StatusInternalServerError {
		t.Fatalf("status = %v, want %d", got, http.StatusInternalServerError)
	}
}

type recordingLogger struct {
	level string
	msg   string
	kv    []any
}

func (r *recordingLogger) Log(level, msg string, kv ...any) {
	r.level, r.msg, r.kv = level, msg, kv
}

func (r *recordingLogger) Logf(string, string, ...any) {}

func (r *recordingLogger) Sync() error { return nil }

type stubMiddlewareContext struct {
	context2.MiddlewareContext
	log context2.LogContext
}

func (s stubMiddlewareContext) Get(string) any { return s.log }

func kvValue(kv []any, key string) any {
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i] == key {
			return kv[i+1]
		}
	}
	return nil
}
