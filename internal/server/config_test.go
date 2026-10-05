package server

import (
	"errors"
	"testing"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

func testConfigStore(t *testing.T, file map[string]any) context2.ConfigContext {
	t.Helper()
	registry := config.NewRegistry()
	RegisterConfig(registry)
	return config.Build(config.Options{Registry: registry, File: file})
}

func TestLoad_SelectsByKind(t *testing.T) {
	cfg := Load(testConfigStore(t, map[string]any{
		"server": map[string]any{
			"type": "rpc",
			"rest": map[string]any{"address": ":1111", "middlewares": []any{"rest-a"}},
			"rpc": map[string]any{
				"address":            ":2222",
				"unary_middlewares":  []any{"rpc-unary-a", "rpc-unary-b"},
				"stream_middlewares": []any{"rpc-stream-a"},
				"reflection":         false,
				"tls": map[string]any{
					"enabled":   true,
					"cert_file": "/etc/tls/server.crt",
					"key_file":  "/etc/tls/server.key",
				},
			},
		},
	}))

	if cfg.Kind != KindRPC || cfg.Address != ":2222" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Reflection || !cfg.TLSEnabled {
		t.Fatalf("rpc reflection/tls = %v/%v, want false/true", cfg.Reflection, cfg.TLSEnabled)
	}
	if cfg.TLSCertFile != "/etc/tls/server.crt" || cfg.TLSKeyFile != "/etc/tls/server.key" {
		t.Fatalf("rpc tls files = %s/%s", cfg.TLSCertFile, cfg.TLSKeyFile)
	}
	if len(cfg.UnaryMiddlewares) != 2 || cfg.UnaryMiddlewares[0] != "rpc-unary-a" {
		t.Fatalf("unary middlewares = %v", cfg.UnaryMiddlewares)
	}
	if len(cfg.StreamMiddlewares) != 1 || cfg.StreamMiddlewares[0] != "rpc-stream-a" {
		t.Fatalf("stream middlewares = %v", cfg.StreamMiddlewares)
	}
	if len(cfg.Middlewares) != 0 {
		t.Fatalf("rest middlewares should stay empty for rpc: %v", cfg.Middlewares)
	}
}

func TestLoad_DefaultsToREST(t *testing.T) {
	cfg := Load(testConfigStore(t, nil))

	if cfg.Kind != KindREST || cfg.Address != DefaultRESTAddress {
		t.Fatalf("cfg = %+v", cfg)
	}
	if len(cfg.Middlewares) != len(defaultRESTMiddlewares) {
		t.Fatalf("middlewares = %v", cfg.Middlewares)
	}
}

func TestNew_UnsupportedKind(t *testing.T) {
	if _, err := New(Config{Kind: "grpc"}, nil, nil); !errors.Is(err, ErrUnsupportedKind) {
		t.Fatalf("err = %v, want ErrUnsupportedKind", err)
	}
}

func TestNew_RESTBuildsServer(t *testing.T) {
	srv, err := New(Config{Kind: KindREST, Address: ":0", Middlewares: []string{"recover"}}, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := srv.(*RestServer); !ok {
		t.Fatalf("server = %T, want *RestServer", srv)
	}
}

func TestNew_RPCBuildsServer(t *testing.T) {
	srv, err := New(Config{
		Kind:              KindRPC,
		Address:           ":0",
		UnaryMiddlewares:  []string{"recover_rpc"},
		StreamMiddlewares: []string{"recover_rpc"},
	}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rpc, ok := srv.(*RpcServer)
	if !ok {
		t.Fatalf("server = %T, want *RpcServer", srv)
	}
	rpc.Setup(context2.ServerContext{})
	if rpc.Handler() == nil {
		t.Fatal("grpc server should be built during Setup")
	}
}
