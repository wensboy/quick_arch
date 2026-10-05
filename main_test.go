package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
	"github.com/wensboy/quick_arch/internal/db"
)

const (
	devInstance    = "dev"
	devDatabaseDSN = "data/store/dev.db"
	devProbeTable  = "crud_probe"
)

type probeRow struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

// TestDevDatabase_Crud 用出厂 config 中的 dev(sqlite) 实例验证上下文加载与完整 CRUD.
func TestDevDatabase_Crud(t *testing.T) {
	ctx := context.Background()

	manager, err := db.Open(ctx, shippedConfig(t))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	// Cleanup 为后进先出: 先清理探针表, 再关闭实例.
	t.Cleanup(func() { _ = manager.Close() })

	var databaseCtx context2.DatabaseContext = manager
	inst, ok := databaseCtx.Instance(devInstance)
	if !ok {
		t.Fatalf("instance %q not loaded, names = %v", devInstance, manager.Names())
	}
	if inst.Driver() != db.DriverSQLite || inst.Kind() != context2.DBKindSQL {
		t.Fatalf("instance = %s/%s, want %s/%s", inst.Driver(), inst.Kind(), db.DriverSQLite, context2.DBKindSQL)
	}
	if _, err := os.Stat(devDatabaseDSN); err != nil {
		t.Fatalf("dev store %s: %v", devDatabaseDSN, err)
	}

	sqlDB, err := databaseCtx.SQL(devInstance)
	if err != nil {
		t.Fatalf("SQL(%s): %v", devInstance, err)
	}
	crud(t, ctx, sqlDB)
}

func shippedConfig(t *testing.T) context2.ConfigContext {
	t.Helper()

	raw, err := config.ReadJSON(&ConfFS, "data/conf/config.json")
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	return config.Build(config.Options{File: raw})
}

func crud(t *testing.T, ctx context.Context, sqlDB *sqlx.DB) {
	t.Helper()

	if _, err := db.Exec(ctx, sqlDB, "DROP TABLE IF EXISTS "+devProbeTable); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if _, err := db.Exec(ctx, sqlDB, "CREATE TABLE "+devProbeTable+" (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, sqlDB, "DROP TABLE IF EXISTS "+devProbeTable)
	})

	if _, err := db.NamedExec(ctx, sqlDB,
		"INSERT INTO "+devProbeTable+" (id, name) VALUES (:id, :name)", probeRow{ID: 1, Name: "alice"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(ctx, sqlDB, "INSERT INTO "+devProbeTable+" (id, name) VALUES (?, ?)", 2, "bob"); err != nil {
		t.Fatalf("create: %v", err)
	}

	one, err := db.Get[probeRow](ctx, sqlDB, "SELECT id, name FROM "+devProbeTable+" WHERE id = ?", 1)
	if err != nil {
		t.Fatalf("read one: %v", err)
	}
	if one.Name != "alice" {
		t.Fatalf("read one = %+v, want alice", one)
	}

	all, err := db.Select[probeRow](ctx, sqlDB, "SELECT id, name FROM "+devProbeTable+" ORDER BY id")
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if len(all) != 2 || all[1].Name != "bob" {
		t.Fatalf("read all = %+v, want [alice bob]", all)
	}

	if _, err := db.Exec(ctx, sqlDB, "UPDATE "+devProbeTable+" SET name = ? WHERE id = ?", "alice2", 1); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, err := db.Get[probeRow](ctx, sqlDB, "SELECT id, name FROM "+devProbeTable+" WHERE id = ?", 1)
	if err != nil {
		t.Fatalf("read updated: %v", err)
	}
	if updated.Name != "alice2" {
		t.Fatalf("updated name = %q, want alice2", updated.Name)
	}

	if _, err := db.Exec(ctx, sqlDB, "DELETE FROM "+devProbeTable+" WHERE id = ?", 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
	remaining, err := db.Select[probeRow](ctx, sqlDB, "SELECT id, name FROM "+devProbeTable)
	if err != nil {
		t.Fatalf("read after delete: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != 1 {
		t.Fatalf("remaining = %+v, want [1]", remaining)
	}

	if _, err := db.Get[probeRow](ctx, sqlDB, "SELECT id, name FROM "+devProbeTable+" WHERE id = ?", 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, want sql.ErrNoRows", err)
	}
}
