package db

import (
	"context"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
)

const (
	DefaultMaxOpenConns = 25
	DefaultMaxIdleConns = 5
	DefaultConnMaxLife  = 30 * time.Minute
)

// SQLInstance 是基于 database/sql 的通用实例, 覆盖各类相似 SQL 数据库.
type SQLInstance struct {
	name   string
	driver string
	db     *sqlx.DB
}

var _ context2.DBInstance = (*SQLInstance)(nil)

func (s *SQLInstance) Name() string                   { return s.name }
func (s *SQLInstance) Driver() string                 { return s.driver }
func (s *SQLInstance) Kind() context2.DBKind          { return context2.DBKindSQL }
func (s *SQLInstance) DB() *sqlx.DB                   { return s.db }
func (s *SQLInstance) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *SQLInstance) Close() error                   { return s.db.Close() }

func openSQL(ctx context.Context, name, driver, dsn string) (*SQLInstance, error) {
	kind, ok := driverKind(driver)
	if !ok {
		return nil, errs.New(ErrUnsupported).With("name", name, "driver", driver)
	}
	if kind != context2.DBKindSQL {
		return nil, errs.New(ErrNotSQL).With("name", name, "driver", driver)
	}

	sqlDB, err := sqlx.Open(driver, dsn)
	if err != nil {
		return nil, errs.Wrap(ErrOpen, err).With("name", name, "driver", driver)
	}
	applyPool(sqlDB, driver)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, errs.Wrap(ErrPing, err).With("name", name, "driver", driver)
	}
	return &SQLInstance{name: name, driver: driver, db: sqlDB}, nil
}

// applyPool sqlite 单文件写入需要串行化, 因此限制单连接.
func applyPool(sqlDB *sqlx.DB, driver string) {
	if driver == DriverSQLite {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		return
	}
	sqlDB.SetMaxOpenConns(DefaultMaxOpenConns)
	sqlDB.SetMaxIdleConns(DefaultMaxIdleConns)
	sqlDB.SetConnMaxLifetime(DefaultConnMaxLife)
}

// Manager 管理全部数据库实例, 实现 context.DatabaseContext.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]context2.DBInstance
	names     []string
}

var _ context2.DatabaseContext = (*Manager)(nil)

func NewManager() *Manager {
	return &Manager{instances: make(map[string]context2.DBInstance)}
}

func (m *Manager) Add(inst context2.DBInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := inst.Name()
	if _, ok := m.instances[name]; ok {
		return errs.New(ErrDuplicate).With("name", name)
	}
	m.instances[name] = inst
	m.names = append(m.names, name)
	return nil
}

func (m *Manager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.names...)
}

func (m *Manager) Drivers() []string { return Drivers() }

func (m *Manager) Instance(name string) (context2.DBInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[name]
	return inst, ok
}

func (m *Manager) SQL(name string) (*sqlx.DB, error) {
	inst, ok := m.Instance(name)
	if !ok {
		return nil, errs.New(ErrNotConfigured).With("name", name)
	}
	sqlInst, ok := inst.(*SQLInstance)
	if !ok {
		return nil, errs.New(ErrNotSQL).With("name", name, "driver", inst.Driver())
	}
	return sqlInst.DB(), nil
}

func (m *Manager) MustSQL(name string) *sqlx.DB {
	sqlDB, err := m.SQL(name)
	if err != nil {
		panic(err)
	}
	return sqlDB
}

func (m *Manager) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, inst := range m.instances {
		if err := inst.Ping(ctx); err != nil {
			return errs.Wrap(ErrPing, err).With("name", name)
		}
	}
	return nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, inst := range m.instances {
		if err := inst.Close(); err != nil && firstErr == nil {
			firstErr = errs.Wrap(ErrClose, err).With("name", name)
		}
	}
	m.instances = make(map[string]context2.DBInstance)
	m.names = nil
	return firstErr
}

// Open 依据 database 列表打开全部实例; 列表为空则不加载任何实例.
func Open(ctx context.Context, cfg context2.ConfigContext) (*Manager, error) {
	specs, err := Load(cfg)
	if err != nil {
		return nil, err
	}

	manager := NewManager()
	for _, spec := range specs {
		inst, err := openSQL(ctx, spec.Name, spec.Driver, spec.DSN)
		if err != nil {
			_ = manager.Close()
			return nil, err
		}
		if err := manager.Add(inst); err != nil {
			_ = manager.Close()
			return nil, err
		}
	}
	return manager, nil
}
