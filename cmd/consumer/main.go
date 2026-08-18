package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"central-pipeline/internal/consumer"
	"central-pipeline/internal/mqtt"
	"central-pipeline/internal/store"
	"central-pipeline/internal/topics"
)

const ClientID = "central-pipeline-consumer"

func main() {
	ctx := context.Background()

	dsn, err := store.DSNFromEnv()
	if err != nil {
		fatal(err)
	}
	db, err := store.Open(ctx, dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	c := &consumer.Consumer{
		Store:    db,
		Timeout:  5 * time.Second,
		Attempts: 5,
		Backoff:  time.Second,
		Fatal:    fatal,
	}

	cfg, err := mqtt.ConfigFromEnv("MQTT_USERNAME", "MQTT_PASSWORD", ClientID, false)
	if err != nil {
		fatal(err)
	}
	cfg.ManualAck = true
	cfg.OnMessage = c.Handle

	client, err := mqtt.Connect(cfg)
	if err != nil {
		fatal(err)
	}
	defer client.Disconnect(250)

	if err := mqtt.Subscribe(client, topics.Subscriptions, 1, c.Handle); err != nil {
		fatal(err)
	}
	slog.Info("consuming", "client_id", ClientID, "topics", topics.Subscriptions)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
}

func fatal(err error) {
	slog.Error(err.Error())
	os.Exit(1)
}
