package mqtt

import (
	"strings"
	"testing"
)

func setAll(t *testing.T) {
	t.Helper()
	t.Setenv("MQTT_HOST", "localhost")
	t.Setenv("MQTT_PORT", "1883")
	t.Setenv("MQTT_USERNAME_SIM", "central-pipeline-sim")
	t.Setenv("MQTT_PASSWORD_SIM", "secret")
}

func TestConfigFromEnv(t *testing.T) {
	setAll(t)
	c, err := ConfigFromEnv("MQTT_USERNAME_SIM", "MQTT_PASSWORD_SIM", "sim-1", true)
	if err != nil {
		t.Fatal(err)
	}
	if c.Host != "localhost" || c.Port != "1883" {
		t.Errorf("broker = %s:%s, want localhost:1883", c.Host, c.Port)
	}
	if c.Username != "central-pipeline-sim" || c.Password != "secret" {
		t.Errorf("credentials = %s/%s", c.Username, c.Password)
	}
	if c.ClientID != "sim-1" || !c.Clean {
		t.Errorf("ClientID = %q, Clean = %v", c.ClientID, c.Clean)
	}
}

// A missing variable must name itself rather than falling back to a default.
func TestConfigFromEnvMissing(t *testing.T) {
	for _, key := range []string{"MQTT_HOST", "MQTT_PORT", "MQTT_USERNAME_SIM", "MQTT_PASSWORD_SIM"} {
		t.Run(key, func(t *testing.T) {
			setAll(t)
			t.Setenv(key, "")
			_, err := ConfigFromEnv("MQTT_USERNAME_SIM", "MQTT_PASSWORD_SIM", "sim-1", true)
			if err == nil {
				t.Fatalf("%s unset but no error", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error %q does not name %s", err, key)
			}
		})
	}
}
