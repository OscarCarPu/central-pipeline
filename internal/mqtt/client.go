package mqtt

import (
	"fmt"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	Host      string
	Port      string
	Username  string
	Password  string
	ClientID  string
	Clean     bool
	ManualAck bool
	OnMessage paho.MessageHandler
}

func ConfigFromEnv(userKey, passKey, clientID string, clean bool) (Config, error) {
	c := Config{
		Host:     os.Getenv("MQTT_HOST"),
		Port:     os.Getenv("MQTT_PORT"),
		Username: os.Getenv(userKey),
		Password: os.Getenv(passKey),
		ClientID: clientID,
		Clean:    clean,
	}
	for _, v := range []struct{ key, val string }{
		{"MQTT_HOST", c.Host},
		{"MQTT_PORT", c.Port},
		{userKey, c.Username},
		{passKey, c.Password},
	} {
		if v.val == "" {
			return Config{}, fmt.Errorf("%s is not set", v.key)
		}
	}
	return c, nil
}

func Connect(c Config) (paho.Client, error) {
	opts := paho.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:%s", c.Host, c.Port)).
		SetClientID(c.ClientID).
		SetUsername(c.Username).
		SetPassword(c.Password).
		SetCleanSession(c.Clean).
		SetConnectTimeout(5 * time.Second).
		SetAutoAckDisabled(c.ManualAck).
		SetAutoReconnect(true).
		SetResumeSubs(true)

	if c.OnMessage != nil {
		opts.SetDefaultPublishHandler(c.OnMessage)
	}

	client := paho.NewClient(opts)
	t := client.Connect()
	if !t.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("connecting to %s:%s timed out", c.Host, c.Port)
	}
	return client, t.Error()
}

func Subscribe(c paho.Client, topics []string, qos byte, h paho.MessageHandler) error {
	filters := make(map[string]byte, len(topics))
	for _, t := range topics {
		filters[t] = qos
	}
	t := c.SubscribeMultiple(filters, h)
	if !t.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("subscribe timed out")
	}
	return t.Error()
}
