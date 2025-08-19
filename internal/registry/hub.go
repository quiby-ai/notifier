package registry

import (
	"sync"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/notifier/config"
)

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

func NewHub(cfg config.Config) *Hub {
	return &Hub{
		cfg:       cfg,
		bySaga:    make(map[string]map[Client]struct{}),
		broadcast: make(chan events.Envelope[events.StateChanged], 1024),
		quit:      make(chan struct{}),
	}
}

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

func (h *Hub) Publish(evt events.Envelope[events.StateChanged]) {
	select {
	case h.broadcast <- evt:
	default:
		// TODO: backpressure: channel full; drop oldest by recreating or just log
		// keep it simple for now (best-effort).
	}
}

func (h *Hub) Close() {
	close(h.quit)
}
