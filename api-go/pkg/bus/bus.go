// Package bus defines an in-process publish/subscribe event bus used for
// inter-module communication so modules never call each other's use cases
// directly. A Redis-backed implementation can satisfy the same Bus
// interface later without changing callers.
package bus

import "sync"

// Event is a named payload published on the bus (e.g. "employee.hired").
type Event struct {
	Name    string
	Payload interface{}
}

// Handler processes a published event.
type Handler func(Event)

// Bus is the publish/subscribe port shared by Core and modules.
type Bus interface {
	Subscribe(name string, handler Handler)
	Publish(event Event)
}

// MemoryBus is a simple synchronous, in-memory Bus implementation.
type MemoryBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewMemoryBus creates an empty in-memory bus.
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{handlers: make(map[string][]Handler)}
}

// Subscribe registers handler to run whenever an event named `name` is published.
func (b *MemoryBus) Subscribe(name string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[name] = append(b.handlers[name], handler)
}

// Publish synchronously invokes every handler subscribed to event.Name.
func (b *MemoryBus) Publish(event Event) {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[event.Name]...)
	b.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

var _ Bus = (*MemoryBus)(nil)
