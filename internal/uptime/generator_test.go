package uptime

import (
	"testing"
	"time"
)

var (
	start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end   = start.Add(30 * 24 * time.Hour)
)

func TestFirstEventIsUp(t *testing.T) {
	g := NewGenerator(1, 0.5)
	for _, d := range Devices {
		e, ok := g.Next(d, start)
		if !ok {
			t.Fatalf("%s: no event on first tick", d)
		}
		if e.State != "up" {
			t.Errorf("%s: first state = %q, want up", d, e.State)
		}
	}
}

func TestStatesAlternate(t *testing.T) {
	events := NewGenerator(7, 0.3).Backfill(start, end, time.Hour)
	if len(events) < 4 {
		t.Fatalf("got %d events, want a stream to check", len(events))
	}
	last := map[Device]string{}
	for _, e := range events {
		if e.State == last[e.Device] {
			t.Fatalf("%s repeated state %q at %s", e.Device, e.State, e.At)
		}
		last[e.Device] = e.State
	}
}

func TestSeedIsDeterministic(t *testing.T) {
	a := NewGenerator(42, 0.1).Backfill(start, end, time.Hour)
	b := NewGenerator(42, 0.1).Backfill(start, end, time.Hour)
	if len(a) != len(b) {
		t.Fatalf("lengths %d and %d differ", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("event %d differs: %+v vs %+v", i, a[i], b[i])
		}
	}
}

func TestRatioBounds(t *testing.T) {
	never := NewGenerator(1, 0).Backfill(start, end, time.Hour)
	if len(never) != len(Devices) {
		t.Errorf("ratio 0 produced %d events, want %d", len(never), len(Devices))
	}
	for _, e := range never {
		if e.State != "up" {
			t.Errorf("ratio 0 produced a %q event", e.State)
		}
	}

	ticks := int(end.Sub(start) / time.Hour)
	always := NewGenerator(1, 1).Backfill(start, end, time.Hour)
	if want := ticks * len(Devices); len(always) != want {
		t.Errorf("ratio 1 produced %d events, want %d", len(always), want)
	}
}

func TestBackfillStaysInRange(t *testing.T) {
	for _, e := range NewGenerator(3, 0.2).Backfill(start, end, time.Hour) {
		if e.At.Before(start) || !e.At.Before(end) {
			t.Errorf("event at %s outside [%s, %s)", e.At, start, end)
		}
	}
}
