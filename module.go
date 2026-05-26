package cachememory

import (
	"context"
	"strings"
	"sync"

	"github.com/Muxcore-Media/core/pkg/contracts"
)

func init() {
	contracts.Register(func(deps contracts.Fabric) contracts.Module {
		return NewModule()
	})
}

type Module struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewModule() *Module {
	return &Module{data: make(map[string][]byte)}
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID:           "cache-memory",
		Name:         "In-Memory Cache",
		Version:      "1.0.0",
		Roles: []string{"provider"},
		Description:  "Simple in-memory read-through cache for the storage orchestrator",
		Author:       "MuxCore",
		Capabilities: []string{"cache.memory", "cache.local"},
	}
}

func (m *Module) Init(ctx context.Context) error  { return nil }
func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }
func (m *Module) Health(ctx context.Context) error { return nil }

func (m *Module) Get(ctx context.Context, key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	return v, ok
}

func (m *Module) Set(ctx context.Context, key string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	return nil
}

func (m *Module) Invalidate(ctx context.Context, prefix string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			delete(m.data, k)
		}
	}
	return nil
}
