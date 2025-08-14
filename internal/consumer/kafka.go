package consumer

import (
	"context"
	"encoding/json"
	"github.com/quiby-ai/notifier/config"
	"github.com/quiby-ai/notifier/internal/registry"
	"github.com/segmentio/kafka-go"
	"log"
)

// KafkaConsumer reads from the configured topic and forwards matching events to the Hub.
type KafkaConsumer struct {
	cfg config.Config
	hub *registry.Hub
}

func NewKafkaConsumer(cfg config.Config, hub *registry.Hub) *KafkaConsumer {
	return &KafkaConsumer{cfg: cfg, hub: hub}
}

func (c *KafkaConsumer) Run(ctx context.Context) {
	rc := kafka.ReaderConfig{
		Brokers:  c.cfg.Kafka.Brokers,
		Topic:    c.cfg.Kafka.Topic,
		MinBytes: 1,
		MaxBytes: 10e6,
	}

	// If GroupID is set, we share work across instances.
	// If empty, we read all messages (fan-out mode) from the beginning.
	if c.cfg.Kafka.GroupID != "" {
		rc.GroupID = c.cfg.Kafka.GroupID
	} else {
		rc.StartOffset = kafka.FirstOffset
	}

	r := kafka.NewReader(rc)
	defer r.Close()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			log.Printf("kafka fetch error: %v", err)
			continue
		}

		var evt registry.KafkaEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("kafka bad json: %v", err)
			_ = r.CommitMessages(ctx, m)
			continue
		}

		// Route only saga state changes (extra guard).
		if evt.Type == "saga.orchestrator.state.changed" && evt.SagaID != "" {
			c.hub.Publish(evt)
		}

		// Best-effort commit; if commit fails, Reader will re-fetch.
		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("kafka commit error: %v", err)
		}
	}
}
