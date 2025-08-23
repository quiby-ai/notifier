package ws

import (
	"context"
	"net/http"
	"time"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/notifier/config"
	"github.com/quiby-ai/notifier/internal/registry"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Client struct {
	conn   *websocket.Conn
	hub    *registry.Hub
	sagaID string
	send   chan events.Envelope[events.StateChanged]
}

func (c *Client) Enqueue(evt events.Envelope[events.StateChanged]) {
	select {
	case c.send <- evt:
	default: /* drop if slow */
	}
}

func WSHandler(h *registry.Hub, cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: check origin

		sagaID := r.URL.Query().Get("saga_id")
		if sagaID == "" {
			http.Error(w, "missing saga_id", http.StatusBadRequest)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			// Do NOT set InsecureSkipVerify; we check Origin above
		})
		if err != nil {
			return
		}

		c := &Client{
			conn: conn, hub: h, sagaID: sagaID,
			send: make(chan events.Envelope[events.StateChanged], 128),
		}
		h.Register(sagaID, c)
		defer func() {
			h.Unregister(sagaID, c)
			_ = conn.Close(websocket.StatusNormalClosure, "bye")
		}()

		go func() {
			pingInt := time.Duration(cfg.WS.PingIntervalSec) * time.Second
			writeTO := time.Duration(cfg.WS.WriteTimeoutSec) * time.Second
			ticker := time.NewTicker(pingInt)
			defer ticker.Stop()
			for {
				select {
				case msg, ok := <-c.send:
					if !ok {
						return
					}
					ctx, cancel := context.WithTimeout(context.Background(), writeTO)
					_ = wsjson.Write(ctx, c.conn, msg)
					cancel()
				case <-ticker.C:
					_ = c.conn.Ping(context.Background())
				}
			}
		}()

		for {
			_, _, err := c.conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	})
}
