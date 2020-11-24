package spec

import "sync"

var registry = struct {
	lock       sync.Mutex
	registered map[string]Spec
}{
	registered: map[string]Spec{},
}

// Register .
func Register(name string, spec Spec) {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	registry.registered[name] = spec
}

// Get .
func Get(name string) Spec {
	registry.lock.Lock()
	defer registry.lock.Unlock()

	return registry.registered[name]
}
