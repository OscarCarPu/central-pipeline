package consumer

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeMessage struct {
	topic   string
	payload []byte
	acks    int
}

func (m *fakeMessage) Duplicate() bool   { return false }
func (m *fakeMessage) Qos() byte         { return 1 }
func (m *fakeMessage) Retained() bool    { return false }
func (m *fakeMessage) Topic() string     { return m.topic }
func (m *fakeMessage) MessageID() uint16 { return 1 }
func (m *fakeMessage) Payload() []byte   { return m.payload }
func (m *fakeMessage) Ack()              { m.acks++ }

type fakeStore struct {
	failures int
	calls    int
}

func (s *fakeStore) InsertEvent(_ context.Context, _ string, _ []byte) error {
	s.calls++
	if s.calls <= s.failures {
		return errors.New("database is down")
	}
	return nil
}

func newConsumer(store Inserter, fatal func(error)) *Consumer {
	return &Consumer{Store: store, Timeout: time.Second, Attempts: 3, Backoff: time.Millisecond, Fatal: fatal}
}

func TestAcksAfterInsert(t *testing.T) {
	store := &fakeStore{}
	msg := &fakeMessage{topic: "events/uptime/lab", payload: []byte(`{"state":"up"}`)}
	newConsumer(store, func(error) { t.Fatal("Fatal called on success") }).Handle(nil, msg)

	if store.calls != 1 {
		t.Errorf("insert called %d times, want 1", store.calls)
	}
	if msg.acks != 1 {
		t.Errorf("acked %d times, want 1", msg.acks)
	}
}

// The broker only redelivers what was never acked, so a permanent failure must
// leave the message unacked.
func TestNeverAcksWhenInsertFails(t *testing.T) {
	store := &fakeStore{failures: 99}
	msg := &fakeMessage{topic: "events/uptime/lab"}
	var fatal error
	newConsumer(store, func(err error) { fatal = err }).Handle(nil, msg)

	if msg.acks != 0 {
		t.Errorf("acked %d times, want 0", msg.acks)
	}
	if store.calls != 3 {
		t.Errorf("insert called %d times, want 3 attempts", store.calls)
	}
	if fatal == nil {
		t.Error("Fatal not called after exhausting attempts")
	}
}

func TestAcksAfterRetry(t *testing.T) {
	store := &fakeStore{failures: 2}
	msg := &fakeMessage{topic: "events/uptime/lab"}
	newConsumer(store, func(err error) { t.Fatalf("Fatal called: %v", err) }).Handle(nil, msg)

	if store.calls != 3 {
		t.Errorf("insert called %d times, want 3", store.calls)
	}
	if msg.acks != 1 {
		t.Errorf("acked %d times, want 1", msg.acks)
	}
}
