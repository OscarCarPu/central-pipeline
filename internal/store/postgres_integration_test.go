//go:build integration

package store

import (
	"context"
	"testing"
	"time"
)

func open(t *testing.T) *Store {
	t.Helper()
	dsn, err := DSNFromEnv()
	if err != nil {
		t.Skipf("no database configured: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := Open(ctx, dsn)
	if err != nil {
		t.Skipf("database unreachable: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestInsertEvent(t *testing.T) {
	s := open(t)
	ctx := context.Background()
	topic := "test/insert-event"

	for _, payload := range [][]byte{
		[]byte(`{"state":"up","timestamp":"2026-08-18T10:00:00Z"}`),
		[]byte("not json"),
	} {
		if err := s.InsertEvent(ctx, topic, payload); err != nil {
			t.Fatalf("InsertEvent(%q): %v", payload, err)
		}
	}
	t.Cleanup(func() {
		s.pool.Exec(context.Background(), "DELETE FROM raw.mqtt_events WHERE topic = $1", topic)
	})

	rows, err := s.pool.Query(ctx,
		"SELECT jsonb_typeof(payload), payload #>> '{}' FROM raw.mqtt_events WHERE topic = $1 ORDER BY id", topic)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got [][2]string
	for rows.Next() {
		var kind, text string
		if err := rows.Scan(&kind, &text); err != nil {
			t.Fatal(err)
		}
		got = append(got, [2]string{kind, text})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	want := [][2]string{{"object", ""}, {"string", "not json"}}
	if len(got) != len(want) {
		t.Fatalf("stored %d rows, want %d", len(got), len(want))
	}
	if got[0][0] != want[0][0] {
		t.Errorf("valid payload stored as %s, want object", got[0][0])
	}
	if got[1][0] != want[1][0] || got[1][1] != want[1][1] {
		t.Errorf("malformed payload stored as %s/%q, want string/%q", got[1][0], got[1][1], want[1][1])
	}
}
