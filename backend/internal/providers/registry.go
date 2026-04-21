package providers

import (
	"fmt"
	"sync"
)

var (
	factoriesMu sync.RWMutex
	factories   = map[Kind]Factory{}
)

func Register(f Factory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[f.Kind()] = f
}

func Lookup(k Kind) (Factory, error) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[k]
	if !ok {
		return nil, fmt.Errorf("unknown feed kind: %s", k)
	}
	return f, nil
}

func All() []Factory {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	out := make([]Factory, 0, len(factories))
	for _, f := range factories {
		out = append(out, f)
	}
	return out
}
