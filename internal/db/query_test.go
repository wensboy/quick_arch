package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
)

type userRow struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

func newSQLite(t *testing.T) *sqlx.DB {
	t.Helper()
	ctx := context.Background()

	manager, err := Open(ctx, testStore(t, []any{sqliteEntry("app", ":memory:")}))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	sqlDB, err := manager.SQL("app")
	if err != nil {
		t.Fatalf("SQL: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return sqlDB
}

func TestNamedExec_Exec_Get_Select(t *testing.T) {
	ctx := context.Background()
	sqlDB := newSQLite(t)

	if _, err := NamedExec(ctx, sqlDB, `INSERT INTO users (id, name) VALUES (:id, :name)`, userRow{ID: 1, Name: "alice"}); err != nil {
		t.Fatalf("NamedExec: %v", err)
	}
	if _, err := Exec(ctx, sqlDB, `INSERT INTO users (id, name) VALUES (?, ?)`, 2, "bob"); err != nil {
		t.Fatalf("Exec: %v", err)
	}

	one, err := Get[userRow](ctx, sqlDB, `SELECT id, name FROM users WHERE id = ?`, 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if one.ID != 1 || one.Name != "alice" {
		t.Fatalf("Get = %+v", one)
	}

	all, err := Select[userRow](ctx, sqlDB, `SELECT id, name FROM users ORDER BY id`)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(all) != 2 || all[1].Name != "bob" {
		t.Fatalf("Select = %+v", all)
	}
}

func TestGet_NotFoundKeepsErrNoRows(t *testing.T) {
	ctx := context.Background()
	if _, err := Get[userRow](ctx, newSQLite(t), `SELECT id, name FROM users WHERE id = ?`, 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err = %v, want sql.ErrNoRows", err)
	}
}

func TestWithTx_CommitAndRollback(t *testing.T) {
	ctx := context.Background()
	sqlDB := newSQLite(t)

	if err := WithTx(ctx, sqlDB, func(tx *sqlx.Tx) error {
		_, err := Exec(ctx, tx, `INSERT INTO users (id, name) VALUES (?, ?)`, 1, "alice")
		return err
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	sentinel := errors.New("boom")
	err := WithTx(ctx, sqlDB, func(tx *sqlx.Tx) error {
		if _, err := Exec(ctx, tx, `INSERT INTO users (id, name) VALUES (?, ?)`, 2, "bob"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}

	count, err := Get[int64](ctx, sqlDB, `SELECT COUNT(*) FROM users`)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if *count != 1 {
		t.Fatalf("count = %d, want 1 after rollback", *count)
	}
}
