package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const insertEvent = `INSERT INTO raw.mqtt_events (topic, payload) VALUES ($1, $2::jsonb)`

type Store struct{ pool *pgxpool.Pool }

func DSNFromEnv() (string, error) {
	v := make(map[string]string, 5)
	for _, k := range []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"} {
		if v[k] = os.Getenv(k); v[k] == "" {
			return "", fmt.Errorf("%s is not set", k)
		}
	}
	// url.URL escapes the userinfo correctly; QueryEscape would turn a space
	// into a +, which is literal in a password.
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(v["POSTGRES_USER"], v["POSTGRES_PASSWORD"]),
		Host:   net.JoinHostPort(v["POSTGRES_HOST"], v["POSTGRES_PORT"]),
		Path:   "/" + v["POSTGRES_DB"],
	}
	return u.String(), nil
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) InsertEvent(ctx context.Context, topic string, payload []byte) error {
	_, err := s.pool.Exec(ctx, insertEvent, topic, JSONPayload(payload))
	return err
}

func JSONPayload(payload []byte) string {
	if json.Valid(payload) {
		return string(payload)
	}
	quoted, err := json.Marshal(string(payload))
	if err != nil {
		return `""`
	}
	return string(quoted)
}
