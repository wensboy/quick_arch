package db

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	errs "github.com/wensboy/quick_arch/internal/error"
)

// Queryer 是 *sqlx.DB 与 *sqlx.Tx 的公共查询子集, 便于同一套代码在事务内外复用.
type Queryer interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

// Get 查询单行并映射到 T; 无结果时保留 sql.ErrNoRows 供 errors.Is 判断.
func Get[T any](ctx context.Context, q Queryer, query string, args ...any) (*T, error) {
	v := new(T)
	if err := q.GetContext(ctx, v, query, args...); err != nil {
		return nil, errs.Wrap(ErrQuery, err)
	}
	return v, nil
}

// Select 查询多行并映射到 []T.
func Select[T any](ctx context.Context, q Queryer, query string, args ...any) ([]T, error) {
	var vs []T
	if err := q.SelectContext(ctx, &vs, query, args...); err != nil {
		return nil, errs.Wrap(ErrQuery, err)
	}
	return vs, nil
}

func Exec(ctx context.Context, q Queryer, query string, args ...any) (sql.Result, error) {
	res, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, errs.Wrap(ErrExec, err)
	}
	return res, nil
}

// NamedExec 执行带命名参数的写操作, SQL 仍为完整语句.
func NamedExec(ctx context.Context, q Queryer, query string, arg any) (sql.Result, error) {
	res, err := q.NamedExecContext(ctx, query, arg)
	if err != nil {
		return nil, errs.Wrap(ErrExec, err)
	}
	return res, err
}

// WithTx 在事务中执行 fn, 返回错误时自动回滚.
func WithTx(ctx context.Context, sqlDB *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := sqlDB.BeginTxx(ctx, nil)
	if err != nil {
		return errs.Wrap(ErrTxBegin, err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return errs.Wrap(ErrTxCommit, err)
	}
	return nil
}
