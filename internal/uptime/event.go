package uptime

import (
	"encoding/json"
	"time"
)

type Device string

const (
	Lab      Device = "lab"
	Watchdog Device = "watchdog"
)

var Devices = []Device{Lab, Watchdog}

type Event struct {
	Device Device
	State  string
	At     time.Time
}

func (e Event) Topic() string { return "events/uptime/" + string(e.Device) }

func (e Event) Payload() ([]byte, error) {
	return json.Marshal(struct {
		State     string `json:"state"`
		Timestamp string `json:"timestamp"`
	}{
		State:     e.State,
		Timestamp: e.At.UTC().Format(time.RFC3339),
	})
}
