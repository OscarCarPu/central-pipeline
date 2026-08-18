package consumer

import (
	"context"
	"log/slog"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Inserter interface {
	InsertEvent(ctx context.Context, topic string, payload []byte) error
}

type Consumer struct {
	Store    Inserter
	Timeout  time.Duration
	Attempts int
	Backoff  time.Duration
	Fatal    func(error)
}

func (c *Consumer) Handle(_ paho.Client, msg paho.Message) {
	var err error
	for attempt := 1; attempt <= c.Attempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		err = c.Store.InsertEvent(ctx, msg.Topic(), msg.Payload())
		cancel()
		if err == nil {
			msg.Ack()
			return
		}
		slog.Warn("insert failed", "topic", msg.Topic(), "attempt", attempt, "error", err)
		time.Sleep(c.Backoff * time.Duration(attempt))
	}
	c.Fatal(err)
}
