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
			"rpc":  map[string]any{"address": ":2222", "middlewares": []any{"rpc-a", "rpc-b"}},
		},
	}))

	if cfg.Kind != KindRPC || cfg.Address != ":2222" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if len(cfg.Middlewares) != 2 || cfg.Middlewares[0] != "rpc-a" {
		t.Fatalf("middlewares = %v", cfg.Middlewares)
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
	if _, err := New(Config{Kind: "grpc"}, nil); !errors.Is(err, ErrUnsupportedKind) {
		t.Fatalf("err = %v, want ErrUnsupportedKind", err)
	}
}

func TestNew_RESTBuildsServer(t *testing.T) {
	srv, err := New(Config{Kind: KindREST, Address: ":0", Middlewares: []string{"recover"}}, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := srv.(*RestServer); !ok {
		t.Fatalf("server = %T, want *RestServer", srv)
	}
}
