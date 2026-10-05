package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
	embed2 "github.com/wensboy/quick_arch/internal/embed"
	errs "github.com/wensboy/quick_arch/internal/error"
	"github.com/wensboy/quick_arch/model"
)

const testSpec = `{"openapi":"3.0.3","info":{"title":"Quick Arch API"},"paths":{"/api/v1/ping":{"get":{"summary":"健康探测"}}}}`

func testConfig(t *testing.T, file map[string]any) context2.ConfigContext {
	t.Helper()
	registry := config.NewRegistry()
	RegisterConfig(registry)
	return config.Build(config.Options{Registry: registry, File: file})
}

func testEmbeds(withSpec bool) context2.EmbedContext {
	store := embed2.NewEmbedStore()
	if withSpec {
		store.SetFile(OpenAPISpecKey, []byte(testSpec))
	}
	return store
}

func newContext(e *echo.Echo, path string) (*echo.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	return e.NewContext(httptest.NewRequest(http.MethodGet, path, nil), rec), rec
}

func TestBuiltinHandler_Ping(t *testing.T) {
	c, rec := newContext(echo.New(), "/api/v1/ping")

	if err := NewBuiltinHandler(nil, nil).Ping(c); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	if resp := decodeResponse(t, rec); resp.Code != model.CodeSuccess || resp.Message != "" {
		t.Fatalf("response = %+v, want success", resp)
	}
	if !strings.Contains(rec.Body.String(), `"data":{"message":"pong"}`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestBuiltinHandler_OpenAPI(t *testing.T) {
	c, rec := newContext(echo.New(), "/api/v1/openapi.json")

	if err := NewBuiltinHandler(testConfig(t, nil), testEmbeds(true)).OpenAPI(c); err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	spec := make(map[string]any)
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	if spec["openapi"] == nil {
		t.Fatalf("spec = %v", spec)
	}
}

func TestBuiltinHandler_Scalar(t *testing.T) {
	c, rec := newContext(echo.New(), "/api/v1/scalar")

	if err := NewBuiltinHandler(testConfig(t, nil), testEmbeds(true)).Scalar(c); err != nil {
		t.Fatalf("Scalar: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{DefaultScalarTitle, "@scalar/api-reference", "/api/v1/ping"} {
		if !strings.Contains(body, want) {
			t.Errorf("scalar page missing %q", want)
		}
	}
}

func TestBuiltinHandler_ScalarDisabled(t *testing.T) {
	handler := NewBuiltinHandler(testConfig(t, map[string]any{
		"services": map[string]any{
			"builtin": map[string]any{"scalar": map[string]any{"enabled": false}},
		},
	}), testEmbeds(true))

	c, _ := newContext(echo.New(), "/api/v1/scalar")
	if err := handler.Scalar(c); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("Scalar err = %v, want ErrNotFound", err)
	}

	c, _ = newContext(echo.New(), "/api/v1/openapi.json")
	if err := handler.OpenAPI(c); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("OpenAPI err = %v, want ErrNotFound", err)
	}
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) model.Response {
	t.Helper()
	var resp model.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return resp
}

func TestBuiltinHandler_SpecMissing(t *testing.T) {
	for name, embeds := range map[string]context2.EmbedContext{
		"nil":     nil,
		"no spec": testEmbeds(false),
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := newContext(echo.New(), "/api/v1/scalar")
			if err := NewBuiltinHandler(testConfig(t, nil), embeds).Scalar(c); err == nil {
				t.Fatal("Scalar err = nil, want error")
			}
		})
	}
}
