package consumer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/notifier/config"
	"github.com/quiby-ai/notifier/internal/registry"
	"github.com/segmentio/kafka-go"
)

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

	if c.cfg.Kafka.GroupID != "" {
		rc.GroupID = c.cfg.Kafka.GroupID
	} else {
		rc.StartOffset = kafka.FirstOffset
	}

	r := kafka.NewReader(rc)
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("kafka reader close error: %v", err)
		}
	}()

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutting down
			}
			log.Printf("kafka fetch error: %v", err)
			continue
		}

		var evt events.Envelope[events.StateChanged]
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("kafka bad json: %v", err)
			_ = r.CommitMessages(ctx, m)
			continue
		}

		if evt.Type == events.SagaStateChanged && evt.SagaID != "" {
			c.hub.Publish(evt)
		}

		if err := r.CommitMessages(ctx, m); err != nil {
			log.Printf("kafka commit error: %v", err)
		}
	}
}
