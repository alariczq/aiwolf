package role

import (
	"fmt"
	"sync"

	"github.com/alaric/eino-learn/internal/config"
)

type Role interface {
	Name() string
	Team() config.Team
	Description() string
	HasNightAction() bool
}

var (
	registry   = make(map[string]Role)
	registryMu sync.RWMutex
)

func Register(r Role) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[r.Name()] = r
}

func Get(name string) (Role, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	r, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown role: %s", name)
	}
	return r, nil
}

func All() map[string]Role {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make(map[string]Role, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
