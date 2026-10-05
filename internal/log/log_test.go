package log

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

func testRegistry() *config.Registry {
	r := config.NewRegistry()
	RegisterConfig(r)
	return r
}

func TestRegister(t *testing.T) {
	for _, e := range testRegistry().Entries() {
		if e.Key != KeyLevel {
			continue
		}
		if e.Flag != "log-level" || e.Env != "LOG_LEVEL" || e.Default != LevelInfo {
			t.Fatalf("log.level entry = %+v", e)
		}
		return
	}
	t.Fatal("log.level entry not registered")
}

func TestLoad_Priority(t *testing.T) {
	t.Setenv("LOG_LEVEL", LevelDebug)

	store := config.Build(config.Options{
		Registry: testRegistry(),
		File:     map[string]any{"log": map[string]any{"level": LevelError, "maxSize": 10}},
		Flags:    map[string]any{"quick_arch.log-level": LevelWarn},
	})

	cfg := Load(store)
	if cfg.Level != LevelWarn {
		t.Errorf("level = %q, want %q", cfg.Level, LevelWarn)
	}
	if cfg.MaxSize != 10 {
		t.Errorf("maxSize = %d, want 10", cfg.MaxSize)
	}
	if cfg.Filename != DefaultFilename {
		t.Errorf("filename = %q, want %q", cfg.Filename, DefaultFilename)
	}
}

func TestLoad_EnvFallsBack(t *testing.T) {
	t.Setenv("LOG_OUTPUT", OutputStderr)
	store := config.Build(config.Options{Registry: testRegistry()})
	if cfg := Load(store); cfg.Output != OutputStderr {
		t.Fatalf("output = %q, want %q", cfg.Output, OutputStderr)
	}
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{LevelDebug, false},
		{strings.ToUpper(LevelInfo), false},
		{" warn ", false},
		{LevelError, false},
		{"", false},
		{"nope", true},
	}
	for _, tc := range cases {
		if _, err := parseLevel(tc.in); (err != nil) != tc.wantErr {
			t.Errorf("parseLevel(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
		}
	}
}

func TestDefaultConfig_Dir(t *testing.T) {
	if got := DefaultConfig().Dir; got != DefaultDir {
		t.Fatalf("default dir = %q, want %q", got, DefaultDir)
	}
}

func TestLogger_LogToFile(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewWithConfig(Config{
		Level:    LevelDebug,
		Encoding: EncodingJSON,
		Output:   OutputFile,
		Dir:      dir,
		Filename: "test.log",
		MaxSize:  DefaultMaxSize,
	})
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}

	logger.Log(LevelInfo, "hello", "k", "v")
	logger.Logf(LevelWarn, "n=%d", 1)
	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	content := readFile(t, filepath.Join(dir, "test.log"))
	if !strings.Contains(content, `"msg":"hello"`) {
		t.Errorf("missing structured log: %s", content)
	}
	if !strings.Contains(content, "n=1") {
		t.Errorf("missing formatted log: %s", content)
	}
}

func TestLogger_LevelFilter(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewWithConfig(Config{
		Level:    LevelWarn,
		Encoding: EncodingConsole,
		Output:   OutputFile,
		Dir:      dir,
		Filename: "filter.log",
		MaxSize:  DefaultMaxSize,
	})
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}

	logger.Log(LevelDebug, "should-be-filtered")
	logger.Log(LevelError, "should-keep")
	_ = logger.Close()

	content := readFile(t, filepath.Join(dir, "filter.log"))
	if strings.Contains(content, "should-be-filtered") {
		t.Errorf("debug log should be filtered: %s", content)
	}
	if !strings.Contains(content, "should-keep") {
		t.Errorf("error log should be kept: %s", content)
	}
}

func TestNewWithConfig_Errors(t *testing.T) {
	if _, err := NewWithConfig(Config{Level: "nope"}); !errors.Is(err, ErrInvalidLevel) {
		t.Errorf("level err = %v, want ErrInvalidLevel", err)
	}
	if _, err := NewWithConfig(Config{Level: LevelInfo, Output: "kafka"}); !errors.Is(err, ErrInvalidOutput) {
		t.Errorf("output err = %v, want ErrInvalidOutput", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return string(b)
}

var _ context2.LogContext = (*Logger)(nil)
