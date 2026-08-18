package uptime

import (
	"math/rand"
	"time"
)

type Generator struct {
	rnd           *rand.Rand
	state         map[Device]string
	downtimeRatio float64
}

func NewGenerator(seed int64, downtimeRatio float64) *Generator {
	g := &Generator{
		rnd:           rand.New(rand.NewSource(seed)),
		state:         make(map[Device]string, len(Devices)),
		downtimeRatio: downtimeRatio,
	}
	for _, d := range Devices {
		g.state[d] = "down"
	}
	return g
}

func (g *Generator) Next(d Device, at time.Time) (Event, bool) {
	next := "up"
	if g.state[d] == "up" {
		if g.rnd.Float64() >= g.downtimeRatio {
			return Event{}, false
		}
		next = "down"
	}
	g.state[d] = next
	return Event{Device: d, State: next, At: at}, true
}

func (g *Generator) Backfill(since, until time.Time, step time.Duration) []Event {
	var out []Event
	for t := since; t.Before(until); t = t.Add(step) {
		for _, d := range Devices {
			if e, ok := g.Next(d, t); ok {
				out = append(out, e)
			}
		}
	}
	return out
}
