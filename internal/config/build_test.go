package config

import (
	"errors"
	"testing"
	"testing/fstest"
)

func testRegistry() *Registry {
	r := NewRegistry()
	r.Register(
		Entry{Key: "log.level", Flag: "log-level", Type: "string", Env: "TEST_LEVEL", Default: "default"},
		Entry{Key: "log.maxSize", Flag: "log-max-size", Type: "int", Default: 100},
		Entry{Key: "log.dir", Env: "TEST_DIR", Default: "data/log"},
	)
	return r
}

func TestBuild_Priority(t *testing.T) {
	t.Setenv("TEST_LEVEL", "env")

	store := Build(Options{
		Registry: testRegistry(),
		File:     map[string]any{"log": map[string]any{"level": "file", "maxSize": 10}},
		Flags:    map[string]any{"quick_arch.log-level": "flag"},
	})

	if v, _ := store.Lookup("log.level"); v != "flag" {
		t.Errorf("log.level = %v, want flag", v)
	}
	if v, _ := store.Lookup("log.maxSize"); v != 10 {
		t.Errorf("log.maxSize = %v, want 10", v)
	}
	if v, _ := store.Lookup("log.dir"); v != "data/log" {
		t.Errorf("log.dir = %v, want data/log", v)
	}
}

func TestBuild_EnvWithoutFlag(t *testing.T) {
	t.Setenv("TEST_LEVEL", "env")
	store := Build(Options{Registry: testRegistry()})
	if v, _ := store.Lookup("log.level"); v != "env" {
		t.Errorf("log.level = %v, want env", v)
	}
}

func TestBuild_IgnoresUnregisteredFlag(t *testing.T) {
	store := Build(Options{
		Registry: testRegistry(),
		Flags:    map[string]any{"quick_arch.verbose": true},
	})
	if _, ok := store.Lookup("verbose"); ok {
		t.Error("unregistered flag should not be mapped")
	}
}

func TestFlatten(t *testing.T) {
	out := make(map[string]any)
	flatten("", map[string]any{
		"$schema": "x",
		"log": map[string]any{
			"level": "info",
			"nested": map[string]any{
				"k": 1,
			},
		},
		"plain": true,
	}, out)

	want := map[string]any{
		"log.level":    "info",
		"log.nested.k": 1,
		"plain":        true,
	}
	if len(out) != len(want) {
		t.Fatalf("flatten size = %d (%v), want %d", len(out), out, len(want))
	}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("flatten[%q] = %v, want %v", k, out[k], v)
		}
	}
}

func TestReadJSON(t *testing.T) {
	fsys := fstest.MapFS{
		"conf/config.json": &fstest.MapFile{
			Data: []byte(`{"log":{"level":"debug"}}`),
		},
	}

	raw, err := ReadJSON(fsys, "conf/config.json")
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	logSection, ok := raw["log"].(map[string]any)
	if !ok || logSection["level"] != "debug" {
		t.Fatalf("ReadJSON = %#v", raw)
	}

	if _, err := ReadJSON(fsys, "conf/missing.json"); !errors.Is(err, ErrReadFile) {
		t.Errorf("err = %v, want ErrReadFile", err)
	}
}
