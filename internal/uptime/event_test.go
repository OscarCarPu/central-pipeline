package uptime

import (
	"testing"
	"time"
)

func TestTopic(t *testing.T) {
	for _, tc := range []struct {
		device Device
		want   string
	}{
		{Lab, "events/uptime/lab"},
		{Watchdog, "events/uptime/watchdog"},
	} {
		if got := (Event{Device: tc.device}).Topic(); got != tc.want {
			t.Errorf("Topic() = %q, want %q", got, tc.want)
		}
	}
}

// The contract in docs/sources/watchdog.md is RFC 3339 UTC to the second.
// time.Time marshals itself as RFC3339Nano, hence the explicit formatting.
func TestPayloadFormat(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	e := Event{
		Device: Lab,
		State:  "up",
		At:     time.Date(2026, 8, 18, 12, 0, 0, 123456789, madrid),
	}
	b, err := e.Payload()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"state":"up","timestamp":"2026-08-18T10:00:00Z"}`
	if string(b) != want {
		t.Errorf("Payload() = %s, want %s", b, want)
	}
}
