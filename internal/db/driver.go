package db

import (
	"sort"

	_ "github.com/mattn/go-sqlite3"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

const (
	DriverSQLite = "sqlite3"
	// DriverMySQL 是 go-sql-driver/mysql 的注册名; DriverMariaDB 为配置层别名, 复用同一驱动.
	DriverMySQL   = "mysql"
	DriverMariaDB = "mariadb"
)

var driverKinds = map[string]context2.DBKind{
	DriverSQLite:  context2.DBKindSQL,
	DriverMySQL:   context2.DBKindSQL,
	DriverMariaDB: context2.DBKindSQL,
}

// RegisterDriver 注册驱动及其类型, 供扩展其它 SQL/NoSQL 数据库.
func RegisterDriver(name string, kind context2.DBKind) {
	driverKinds[name] = kind
}

func Drivers() []string {
	names := make([]string, 0, len(driverKinds))
	for name := range driverKinds {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func driverKind(name string) (context2.DBKind, bool) {
	kind, ok := driverKinds[name]
	return kind, ok
}
