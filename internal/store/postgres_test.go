package store

import (
	"strings"
	"testing"
)

func TestJSONPayload(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    string
	}{
		{"object", `{"state":"up","timestamp":"2026-08-18T10:00:00Z"}`, `{"state":"up","timestamp":"2026-08-18T10:00:00Z"}`},
		{"scalar", `42`, `42`},
		{"not json", `not json`, `"not json"`},
		{"empty", ``, `""`},
		{"truncated", `{"state":"up"`, `"{\"state\":\"up\""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := JSONPayload([]byte(tc.payload)); got != tc.want {
				t.Errorf("JSONPayload(%q) = %s, want %s", tc.payload, got, tc.want)
			}
		})
	}
}

func setDB(t *testing.T) {
	t.Helper()
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "user")
	t.Setenv("POSTGRES_PASSWORD", "pass word/@")
	t.Setenv("POSTGRES_DB", "db")
}

func TestDSNFromEnv(t *testing.T) {
	setDB(t)
	dsn, err := DSNFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	want := "postgres://user:pass%20word%2F%40@localhost:5432/db"
	if dsn != want {
		t.Errorf("DSNFromEnv() = %s, want %s", dsn, want)
	}
}

func TestDSNFromEnvMissing(t *testing.T) {
	for _, key := range []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"} {
		t.Run(key, func(t *testing.T) {
			setDB(t)
			t.Setenv(key, "")
			_, err := DSNFromEnv()
			if err == nil {
				t.Fatalf("%s unset but no error", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("error %q does not name %s", err, key)
			}
		})
	}
}
