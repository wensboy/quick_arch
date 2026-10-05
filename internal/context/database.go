package context

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type DBKind string

const (
	DBKindSQL   DBKind = "sql"
	DBKindNoSQL DBKind = "nosql"
)

// DBInstance 是数据库实例的窄接口, 只描述生命周期与元信息.
// 差异过大的数据库 (如各类 NoSQL) 只需实现该接口即可纳入管理.
type DBInstance interface {
	Name() string
	Driver() string
	Kind() DBKind
	Ping(ctx context.Context) error
	Close() error
}

type DatabaseContext interface {
	// Drivers 返回已支持的驱动名.
	Drivers() []string
	// Instance 返回指定实例的窄接口句柄.
	Instance(name string) (DBInstance, bool)
	// SQL 返回指定 SQL 实例的 sqlx 句柄.
	SQL(name string) (*sqlx.DB, error)
	MustSQL(name string) *sqlx.DB
}
