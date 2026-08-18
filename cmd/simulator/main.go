package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"central-pipeline/internal/mqtt"
	"central-pipeline/internal/uptime"
)

type publisher struct {
	client     paho.Client
	rnd        *rand.Rand
	duplicates float64
	skew       time.Duration
}

func main() {
	mode := flag.String("mode", "live", "live, backfill or once")
	device := flag.String("device", "lab", "device, for -mode=once")
	state := flag.String("state", "up", "state, for -mode=once")
	interval := flag.Duration("interval", 5*time.Second, "time between live ticks")
	since := flag.Duration("since", 365*24*time.Hour, "how far back to backfill")
	step := flag.Duration("step", time.Hour, "simulated time between backfill ticks")
	seed := flag.Int64("seed", 1, "generator seed")
	ratio := flag.Float64("downtime-ratio", 0.02, "chance an up device goes down")
	duplicates := flag.Float64("duplicates", 0, "chance an event is published twice")
	skew := flag.Duration("skew", 0, "shift duplicated timestamps backwards")
	malformed := flag.Bool("malformed", false, "publish one invalid payload and exit")
	flag.Parse()

	clientID := fmt.Sprintf("central-pipeline-sim-%d", time.Now().UnixNano()%1e6)
	cfg, err := mqtt.ConfigFromEnv("MQTT_USERNAME_SIM", "MQTT_PASSWORD_SIM", clientID, true)
	if err != nil {
		log.Fatal(err)
	}
	client, err := mqtt.Connect(cfg)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Disconnect(250)

	p := &publisher{
		client:     client,
		rnd:        rand.New(rand.NewSource(*seed)),
		duplicates: *duplicates,
		skew:       *skew,
	}

	if *malformed {
		if err := p.raw(uptime.Event{Device: uptime.Lab}.Topic(), []byte("not json")); err != nil {
			log.Fatal(err)
		}
		log.Print("published one malformed payload")
		return
	}

	switch *mode {
	case "once":
		e := uptime.Event{Device: uptime.Device(*device), State: *state, At: time.Now()}
		if err := p.send(e); err != nil {
			log.Fatal(err)
		}
		log.Printf("%s %s", e.Topic(), e.State)

	case "backfill":
		until := time.Now()
		events := uptime.NewGenerator(*seed, *ratio).Backfill(until.Add(-*since), until, *step)
		for _, e := range events {
			if err := p.send(e); err != nil {
				log.Fatal(err)
			}
		}
		log.Printf("published %d events over %s", len(events), *since)

	case "live":
		g := uptime.NewGenerator(*seed, *ratio)
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		for {
			select {
			case <-stop:
				return
			case now := <-ticker.C:
				for _, d := range uptime.Devices {
					e, ok := g.Next(d, now)
					if !ok {
						continue
					}
					if err := p.send(e); err != nil {
						log.Print(err)
						continue
					}
					log.Printf("%s %s", e.Topic(), e.State)
				}
			}
		}

	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func (p *publisher) send(e uptime.Event) error {
	b, err := e.Payload()
	if err != nil {
		return err
	}
	if err := p.raw(e.Topic(), b); err != nil {
		return err
	}
	if p.duplicates <= 0 || p.rnd.Float64() >= p.duplicates {
		return nil
	}
	dup := e
	dup.At = e.At.Add(-p.skew)
	if b, err = dup.Payload(); err != nil {
		return err
	}
	return p.raw(dup.Topic(), b)
}

func (p *publisher) raw(topic string, payload []byte) error {
	t := p.client.Publish(topic, 1, false, payload)
	if !t.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("publish to %s timed out", topic)
	}
	return t.Error()
}
