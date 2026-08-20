package main

import (
	"testing"
	"time"
)

func TestEventTime(t *testing.T) {
	before := time.Now()
	got, err := eventTime("")
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if got.Before(before) {
		t.Errorf("empty -at = %s, want now or later", got)
	}

	got, err = eventTime("2026-08-19T13:00:00Z")
	if err != nil {
		t.Fatalf("valid: %v", err)
	}
	if want := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("got %s, want %s", got, want)
	}

	// A local-time offset must land on the same instant, not the same clock face.
	got, err = eventTime("2026-08-19T15:00:00+02:00")
	if err != nil {
		t.Fatalf("offset: %v", err)
	}
	if want := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Errorf("offset got %s, want %s", got, want)
	}

	for _, bad := range []string{"yesterday", "2026-08-19", "2026-08-19 13:00:00"} {
		if _, err := eventTime(bad); err == nil {
			t.Errorf("eventTime(%q) = nil error, want failure", bad)
		}
	}
}
