package server

import (
	"sort"
	"sync"

	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

type MiddlewareStore struct {
	mu        sync.RWMutex
	values    map[string]any
	factories map[string]context2.MiddlewareFactory
	resolved  map[string][]echo.MiddlewareFunc
}

var _ context2.MiddlewareContext = (*MiddlewareStore)(nil)

func NewMiddlewareStore() *MiddlewareStore {
	return &MiddlewareStore{
		values:    make(map[string]any),
		factories: make(map[string]context2.MiddlewareFactory),
		resolved:  make(map[string][]echo.MiddlewareFunc),
	}
}

// Bind 将第三方模块上下文注入值区, 供中间件工厂读取.
func (m *MiddlewareStore) Bind(log context2.LogContext, cfg context2.ConfigContext, db context2.DatabaseContext) *MiddlewareStore {
	m.Set(context2.MiddlewareValueLog, log)
	m.Set(context2.MiddlewareValueConfig, cfg)
	m.Set(context2.MiddlewareValueDB, db)
	return m
}

func (m *MiddlewareStore) Set(key string, value any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
}

func (m *MiddlewareStore) Get(key string) any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.values[key]
}

func (m *MiddlewareStore) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.values[key]
	return ok
}

func (m *MiddlewareStore) Del(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.values, key)
}

func (m *MiddlewareStore) Register(name string, factory context2.MiddlewareFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.factories[name] = factory
	delete(m.resolved, name)
}

func (m *MiddlewareStore) Registered() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.factories))
	for name := range m.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Resolve 按名称顺序解析中间件; 未注册的名称会被忽略.
// 工厂在首次解析时执行并缓存, 结果可被多个路由复用.
func (m *MiddlewareStore) Resolve(names ...string) []echo.MiddlewareFunc {
	var chain []echo.MiddlewareFunc
	for _, name := range names {
		if built, ok := m.cached(name); ok {
			chain = append(chain, built...)
			continue
		}
		factory, ok := m.factory(name)
		if !ok {
			continue
		}

		built := factory(m)
		m.cache(name, built)
		chain = append(chain, built...)
	}
	return chain
}

func (m *MiddlewareStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resolved = make(map[string][]echo.MiddlewareFunc)
}

func (m *MiddlewareStore) cached(name string) ([]echo.MiddlewareFunc, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	built, ok := m.resolved[name]
	return built, ok
}

func (m *MiddlewareStore) factory(name string) (context2.MiddlewareFactory, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	factory, ok := m.factories[name]
	return factory, ok
}

func (m *MiddlewareStore) cache(name string, built []echo.MiddlewareFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resolved[name] = built
}
