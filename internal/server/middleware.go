package server

import (
	"sort"
	"sync"

	"github.com/labstack/echo/v5"
	"google.golang.org/grpc"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

type MiddlewareStore struct {
	mu              sync.RWMutex
	values          map[string]any
	factories       map[string]context2.MiddlewareFactory
	resolved        map[string][]echo.MiddlewareFunc
	unaryFactories  map[string]context2.UnaryInterceptorFactory
	unaryResolved   map[string][]grpc.UnaryServerInterceptor
	streamFactories map[string]context2.StreamInterceptorFactory
	streamResolved  map[string][]grpc.StreamServerInterceptor
}

var _ context2.MiddlewareContext = (*MiddlewareStore)(nil)

func NewMiddlewareStore() *MiddlewareStore {
	return &MiddlewareStore{
		values:          make(map[string]any),
		factories:       make(map[string]context2.MiddlewareFactory),
		resolved:        make(map[string][]echo.MiddlewareFunc),
		unaryFactories:  make(map[string]context2.UnaryInterceptorFactory),
		unaryResolved:   make(map[string][]grpc.UnaryServerInterceptor),
		streamFactories: make(map[string]context2.StreamInterceptorFactory),
		streamResolved:  make(map[string][]grpc.StreamServerInterceptor),
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

func (m *MiddlewareStore) RegisterUnary(name string, factory context2.UnaryInterceptorFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unaryFactories[name] = factory
	delete(m.unaryResolved, name)
}

func (m *MiddlewareStore) RegisterStream(name string, factory context2.StreamInterceptorFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamFactories[name] = factory
	delete(m.streamResolved, name)
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

// ResolveUnary 是 Resolve 的 gRPC 一元形态, 语义一致.
func (m *MiddlewareStore) ResolveUnary(names ...string) []grpc.UnaryServerInterceptor {
	var chain []grpc.UnaryServerInterceptor
	for _, name := range names {
		if built, ok := m.unaryCached(name); ok {
			chain = append(chain, built...)
			continue
		}
		factory, ok := m.unaryFactory(name)
		if !ok {
			continue
		}

		built := factory(m)
		m.unaryCache(name, built)
		chain = append(chain, built...)
	}
	return chain
}

// ResolveStream 是 Resolve 的 gRPC 流形态, 语义一致.
func (m *MiddlewareStore) ResolveStream(names ...string) []grpc.StreamServerInterceptor {
	var chain []grpc.StreamServerInterceptor
	for _, name := range names {
		if built, ok := m.streamCached(name); ok {
			chain = append(chain, built...)
			continue
		}
		factory, ok := m.streamFactory(name)
		if !ok {
			continue
		}

		built := factory(m)
		m.streamCache(name, built)
		chain = append(chain, built...)
	}
	return chain
}

func (m *MiddlewareStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resolved = make(map[string][]echo.MiddlewareFunc)
	m.unaryResolved = make(map[string][]grpc.UnaryServerInterceptor)
	m.streamResolved = make(map[string][]grpc.StreamServerInterceptor)
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

func (m *MiddlewareStore) unaryCached(name string) ([]grpc.UnaryServerInterceptor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	built, ok := m.unaryResolved[name]
	return built, ok
}

func (m *MiddlewareStore) unaryFactory(name string) (context2.UnaryInterceptorFactory, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	factory, ok := m.unaryFactories[name]
	return factory, ok
}

func (m *MiddlewareStore) unaryCache(name string, built []grpc.UnaryServerInterceptor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unaryResolved[name] = built
}

func (m *MiddlewareStore) streamCached(name string) ([]grpc.StreamServerInterceptor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	built, ok := m.streamResolved[name]
	return built, ok
}

func (m *MiddlewareStore) streamFactory(name string) (context2.StreamInterceptorFactory, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	factory, ok := m.streamFactories[name]
	return factory, ok
}

func (m *MiddlewareStore) streamCache(name string, built []grpc.StreamServerInterceptor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamResolved[name] = built
}
