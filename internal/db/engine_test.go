package db

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

func testStore(t *testing.T, database []any) *config.ConfigStore {
	t.Helper()
	return config.Build(config.Options{File: map[string]any{KeyDatabase: database}})
}

func sqliteEntry(name, dsn string) map[string]any {
	return map[string]any{"name": name, "driver": DriverSQLite, "dsn": dsn}
}

func mysqlEntry(overrides map[string]any) map[string]any {
	entry := map[string]any{
		"name":     "app",
		"driver":   DriverMySQL,
		"host":     "127.0.0.1",
		"user":     "root",
		"database": "quick_arch",
	}
	for k, v := range overrides {
		entry[k] = v
	}
	return entry
}

func TestOpen_EmptyListLoadsNothing(t *testing.T) {
	manager, err := Open(context.Background(), testStore(t, nil))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = manager.Close() }()

	if got := manager.Names(); len(got) != 0 {
		t.Fatalf("names = %v, want empty", got)
	}
}

func TestOpen_LoadsConfiguredInstance(t *testing.T) {
	ctx := context.Background()
	manager, err := Open(ctx, testStore(t, []any{sqliteEntry("app", ":memory:")}))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = manager.Close() }()

	if got := manager.Names(); len(got) != 1 || got[0] != "app" {
		t.Fatalf("names = %v, want [app]", got)
	}
	if _, ok := manager.Instance("other"); ok {
		t.Error("unknown instance should not exist")
	}
	if err := manager.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if _, err := manager.SQL("app"); err != nil {
		t.Fatalf("SQL: %v", err)
	}
}

func TestOpen_DuplicateNameSkipsLater(t *testing.T) {
	manager, err := Open(context.Background(), testStore(t, []any{
		sqliteEntry("app", ":memory:"),
		sqliteEntry("app", "/no/such/dir/app.db"),
	}))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = manager.Close() }()

	if got := manager.Names(); len(got) != 1 || got[0] != "app" {
		t.Fatalf("names = %v, want [app]", got)
	}
}

func TestOpen_UnsupportedDriver(t *testing.T) {
	_, err := Open(context.Background(), testStore(t, []any{
		map[string]any{"name": "app", "driver": "kafka", "dsn": "addr=1"},
	}))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestOpen_InvalidInstance(t *testing.T) {
	cases := []struct {
		name  string
		entry map[string]any
	}{
		{"missing name", map[string]any{"driver": DriverSQLite, "dsn": ":memory:"}},
		{"missing dsn", map[string]any{"name": "app", "driver": DriverSQLite}},
		{"mysql missing host", mysqlEntry(map[string]any{"host": ""})},
		{"mysql bad net", mysqlEntry(map[string]any{"net": "udp"})},
		{"mysql bad loc", mysqlEntry(map[string]any{"loc": "Mars/Olympus"})},
		{"mysql bad timeout", mysqlEntry(map[string]any{"timeout": "5"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Open(context.Background(), testStore(t, []any{tc.entry})); !errors.Is(err, ErrInstanceConfig) {
				t.Fatalf("err = %v, want ErrInstanceConfig", err)
			}
		})
	}
}

func TestLoad_MySQLAttributes(t *testing.T) {
	specs, err := Load(testStore(t, []any{
		mysqlEntry(map[string]any{
			"name":              "report",
			"host":              "db.internal",
			"port":              3307,
			"password":          "secret",
			"charset":           "utf8mb4",
			"collation":         "utf8mb4_general_ci",
			"parseTime":         true,
			"loc":               "Local",
			"timeout":           "5s",
			"readTimeout":       "10s",
			"writeTimeout":      "10s",
			"tls":               "preferred",
			"multiStatements":   true,
			"interpolateParams": true,
			"params":            map[string]any{"time_zone": "+08:00"},
		}),
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("specs = %+v", specs)
	}

	dsn := specs[0].DSN
	for _, want := range []string{
		"root:secret@tcp(db.internal:3307)/quick_arch",
		"charset=utf8mb4",
		"collation=utf8mb4_general_ci",
		"parseTime=true",
		"loc=Local",
		"timeout=5s",
		"readTimeout=10s",
		"writeTimeout=10s",
		"tls=preferred",
		"multiStatements=true",
		"interpolateParams=true",
		"time_zone=",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn = %q, missing %q", dsn, want)
		}
	}
}

func TestLoad_MariaDBUsesMySQLDriver(t *testing.T) {
	specs, err := Load(testStore(t, []any{
		mysqlEntry(map[string]any{"name": "maria", "driver": DriverMariaDB}),
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if specs[0].Driver != DriverMySQL {
		t.Fatalf("driver = %q, want %q", specs[0].Driver, DriverMySQL)
	}
	if !strings.Contains(specs[0].DSN, "tcp(127.0.0.1:3306)/quick_arch") {
		t.Fatalf("dsn = %q", specs[0].DSN)
	}
}

func TestLoad_MySQLUnixSocket(t *testing.T) {
	specs, err := Load(testStore(t, []any{
		mysqlEntry(map[string]any{"net": "unix", "host": "/run/mysqld/mysqld.sock"}),
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(specs[0].DSN, "unix(/run/mysqld/mysqld.sock)/quick_arch") {
		t.Fatalf("dsn = %q", specs[0].DSN)
	}
}

func TestLoad_SQLiteKeepsDsn(t *testing.T) {
	specs, err := Load(testStore(t, []any{sqliteEntry("app", ":memory:")}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if specs[0].DSN != ":memory:" || specs[0].Driver != DriverSQLite {
		t.Fatalf("spec = %+v", specs[0])
	}
}

func TestSQL_NotConfigured(t *testing.T) {
	if _, err := NewManager().SQL("nope"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

func TestSQL_NonSQLInstance(t *testing.T) {
	manager := NewManager()
	if err := manager.Add(&stubInstance{name: "kv", driver: "redis", kind: context2.DBKindNoSQL}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := manager.SQL("kv"); !errors.Is(err, ErrNotSQL) {
		t.Fatalf("err = %v, want ErrNotSQL", err)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	manager := NewManager()
	inst := &stubInstance{name: "x", driver: DriverSQLite, kind: context2.DBKindSQL}
	if err := manager.Add(inst); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := manager.Add(inst); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("err = %v, want ErrDuplicate", err)
	}
}

func TestDrivers(t *testing.T) {
	set := make(map[string]bool)
	for _, name := range Drivers() {
		set[name] = true
	}
	for _, want := range []string{DriverSQLite, DriverMySQL, DriverMariaDB} {
		if !set[want] {
			t.Fatalf("drivers = %v, missing %s", Drivers(), want)
		}
	}
}

type stubInstance struct {
	name   string
	driver string
	kind   context2.DBKind
}

func (s *stubInstance) Name() string               { return s.name }
func (s *stubInstance) Driver() string             { return s.driver }
func (s *stubInstance) Kind() context2.DBKind      { return s.kind }
func (s *stubInstance) Ping(context.Context) error { return nil }
func (s *stubInstance) Close() error               { return nil }
