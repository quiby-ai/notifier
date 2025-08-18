package registry

import (
	"sync"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/notifier/config"
)

// Client interface defines what a WebSocket client must implement
type Client interface {
	Enqueue(evt events.Envelope[events.StateChanged])
}

type Hub struct {
	cfg config.Config

	mu     sync.RWMutex
	bySaga map[string]map[Client]struct{} // saga_id -> clients

	broadcast chan events.Envelope[events.StateChanged]
	quit      chan struct{}
}

// NewHub constructs the in-memory connection registry and broadcast loop.
func NewHub(cfg config.Config) *Hub {
	return &Hub{
		cfg:       cfg,
		bySaga:    make(map[string]map[Client]struct{}),
		broadcast: make(chan events.Envelope[events.StateChanged], 1024),
		quit:      make(chan struct{}),
	}
}

// Run handles fanout for published events until Close() is called.
func (h *Hub) Run() {
	for {
		select {
		case evt := <-h.broadcast:
			h.mu.RLock()
			conns := h.bySaga[evt.SagaID]
			for c := range conns {
				c.Enqueue(evt) // non-blocking; drops if client is slow
			}
			h.mu.RUnlock()
		case <-h.quit:
			return
		}
	}
}

// Register adds a client under a saga_id.
func (h *Hub) Register(sagaID string, c Client) {
	h.mu.Lock()
	set, ok := h.bySaga[sagaID]
	if !ok {
		set = make(map[Client]struct{})
		h.bySaga[sagaID] = set
	}
	set[c] = struct{}{}
	h.mu.Unlock()
}

// Unregister removes a client from a saga_id set.
func (h *Hub) Unregister(sagaID string, c Client) {
	h.mu.Lock()
	if set, ok := h.bySaga[sagaID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.bySaga, sagaID)
		}
	}
	h.mu.Unlock()
}

// Publish queues an event for fanout to all clients on that saga_id.
func (h *Hub) Publish(evt events.Envelope[events.StateChanged]) {
	select {
	case h.broadcast <- evt:
	default:
		// backpressure: channel full; drop oldest by recreating or just log
		// keep it simple for now (best-effort).
	}
}

func (h *Hub) Close() {
	close(h.quit)
}
