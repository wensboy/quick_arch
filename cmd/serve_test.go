package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/wensboy/quick_arch/handler"
	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
	"github.com/wensboy/quick_arch/internal/db"
	embed2 "github.com/wensboy/quick_arch/internal/embed"
	"github.com/wensboy/quick_arch/internal/server"
	"github.com/wensboy/quick_arch/model"
	builtinpb "github.com/wensboy/quick_arch/proto/builtin"
)

const testOpenAPISpec = `{"openapi":"3.0.3","paths":{"/api/v1/ping":{"get":{}}}}`

func testAppContext(t *testing.T, file map[string]any) context2.AppContext {
	t.Helper()

	embeds := embed2.NewEmbedStore()
	embeds.SetFile(handler.OpenAPISpecKey, []byte(testOpenAPISpec))

	return context2.AppContext{
		Context:       context.Background(),
		EmbedContext:  embeds,
		ConfigContext: config.Build(config.Options{Registry: newRegistry(), File: file}),
	}
}

func serveRequest(t *testing.T, app context2.AppContext, path string) *httptest.ResponseRecorder {
	t.Helper()

	srv, _, cleanup, err := setupServe(app)
	if err != nil {
		t.Fatalf("setupServe: %v", err)
	}
	defer cleanup()

	rest, ok := srv.(*server.RestServer)
	if !ok {
		t.Fatalf("server = %T, want *server.RestServer", srv)
	}

	rec := httptest.NewRecorder()
	rest.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestServe_PingChain(t *testing.T) {
	app := testAppContext(t, map[string]any{
		"server": map[string]any{
			"type": "rest",
			"rest": map[string]any{
				"address":     ":0",
				"middlewares": []any{"access_log", "recover", "request_id", "debug"},
			},
		},
		"middleware": map[string]any{
			"debug": map[string]any{"include": []any{"/api"}},
		},
	})

	rec := serveRequest(t, app, "/api/v1/ping")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("request_id middleware not applied")
	}
	if rec.Header().Get("X-Debug-Middleware") != "debug" {
		t.Error("debug middleware not applied")
	}

	var body pingEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if body.Code != model.CodeSuccess || body.Data.Message != "pong" {
		t.Fatalf("body = %+v, want success/pong", body)
	}
}

type pingEnvelope struct {
	Code int32 `json:"code"`
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

func TestServe_MiddlewareExclude(t *testing.T) {
	app := testAppContext(t, map[string]any{
		"server": map[string]any{
			"type": "rest",
			"rest": map[string]any{"middlewares": []any{"debug"}},
		},
		"middleware": map[string]any{
			"debug": map[string]any{"exclude": []any{"/api/v1/ping"}},
		},
	})

	rec := serveRequest(t, app, "/api/v1/ping")

	if rec.Header().Get("X-Debug-Middleware") != "" {
		t.Fatalf("debug middleware should be excluded, headers = %v", rec.Header())
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestServe_UnsupportedKind(t *testing.T) {
	app := testAppContext(t, map[string]any{
		"server": map[string]any{"type": "graphql"},
	})

	if _, _, _, err := setupServe(app); !errors.Is(err, server.ErrUnsupportedKind) {
		t.Fatalf("err = %v, want ErrUnsupportedKind", err)
	}
}

// TestServe_RPCPingOverBufconn 用真实 gRPC 调用覆盖 proto -> 实现 -> 服务端装配 全链路.
func TestServe_RPCPingOverBufconn(t *testing.T) {
	app := testAppContext(t, map[string]any{
		"server": map[string]any{
			"type": "rpc",
			"rpc": map[string]any{
				"address":            ":0",
				"unary_middlewares":  []any{"recover_rpc"},
				"stream_middlewares": []any{"recover_rpc"},
			},
		},
	})

	srv, _, cleanup, err := setupServe(app)
	if err != nil {
		t.Fatalf("setupServe: %v", err)
	}
	defer cleanup()

	rpc, ok := srv.(*server.RpcServer)
	if !ok {
		t.Fatalf("server = %T, want *server.RpcServer", srv)
	}

	listener := bufconn.Listen(1024 * 1024)
	go func() { _ = rpc.Handler().Serve(listener) }()
	defer rpc.Handler().Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	defer func() { _ = conn.Close() }()

	resp, err := builtinpb.NewBuiltinClient(conn).Ping(context.Background(), &builtinpb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if resp.GetMessage() != "pong" {
		t.Fatalf("message = %q, want pong", resp.GetMessage())
	}
}

func TestServe_RPCBuildsRpcServer(t *testing.T) {
	app := testAppContext(t, map[string]any{
		"server": map[string]any{
			"type": "rpc",
			"rpc": map[string]any{
				"address":            ":0",
				"unary_middlewares":  []any{"recover_rpc"},
				"stream_middlewares": []any{"recover_rpc"},
			},
		},
	})

	srv, _, cleanup, err := setupServe(app)
	if err != nil {
		t.Fatalf("setupServe: %v", err)
	}
	defer cleanup()

	rpc, ok := srv.(*server.RpcServer)
	if !ok {
		t.Fatalf("server = %T, want *server.RpcServer", srv)
	}
	if rpc.Handler() == nil {
		t.Fatal("grpc server should be built during setup")
	}
}

// TestServe_PingOverHTTP 走真实 socket, 覆盖 中间件 -> 路由 -> handler 全链路.
func TestServe_PingOverHTTP(t *testing.T) {
	app := testAppContext(t, map[string]any{
		"server": map[string]any{
			"type": "rest",
			"rest": map[string]any{
				"middlewares": []any{"access_log", "recover", "request_id", "debug"},
			},
		},
	})

	srv, _, cleanup, err := setupServe(app)
	if err != nil {
		t.Fatalf("setupServe: %v", err)
	}
	defer cleanup()

	ts := httptest.NewServer(srv.(*server.RestServer).Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/ping")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Request-Id") == "" || resp.Header.Get("X-Debug-Middleware") != "debug" {
		t.Errorf("middleware headers missing: %v", resp.Header)
	}

	var body pingEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != model.CodeSuccess || body.Data.Message != "pong" {
		t.Fatalf("body = %+v, want success/pong", body)
	}
}

// TestServe_UnknownRouteUnifiedError 校验未命中路由也走统一响应格式.
func TestServe_UnknownRouteUnifiedError(t *testing.T) {
	rec := serveRequest(t, testAppContext(t, nil), "/api/v1/nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"code":20001,"message":"资源不存在"}` {
		t.Fatalf("body = %s", got)
	}
}

func TestServe_DatabaseContextLoadsInstance(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "dev.db")
	app := testAppContext(t, map[string]any{
		"database": []any{
			map[string]any{"name": "dev", "driver": "sqlite3", "dsn": dsn},
		},
	})

	_, sctx, cleanup, err := setupServe(app)
	if err != nil {
		t.Fatalf("setupServe: %v", err)
	}
	defer cleanup()

	if _, ok := sctx.Instance("dev"); !ok {
		t.Fatal("dev instance not wired into server context")
	}
	sqlDB, err := sctx.SQL("dev")
	if err != nil {
		t.Fatalf("SQL(dev): %v", err)
	}
	if _, err := db.Exec(context.Background(), sqlDB, "CREATE TABLE probe (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("exec on dev: %v", err)
	}
}

func TestServe_ScalarEndpoint(t *testing.T) {
	rec := serveRequest(t, testAppContext(t, nil), "/api/v1/scalar")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "@scalar/api-reference") {
		t.Fatal("scalar page missing api-reference script")
	}
}

func TestServe_OpenAPIEndpoint(t *testing.T) {
	rec := serveRequest(t, testAppContext(t, nil), "/api/v1/openapi.json")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"/api/v1/ping"`) {
		t.Fatal("openapi spec missing ping path")
	}
}
