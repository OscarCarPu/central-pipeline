up:
	docker compose up -d --wait

down:
	docker compose down

restart:
	docker compose down -v
	docker compose up -d --build --wait

test:
	go test ./...

simulate:
	@set -a; . ./.env; set +a; go run ./cmd/simulator $(ARGS)

consume:
	@set -a; . ./.env; set +a; go run ./cmd/consumer

test-integration:
	@set -a; . ./.env; set +a; go test -tags=integration ./...

db:
	@set -a; . ./.env; set +a; PGPASSWORD=$$POSTGRES_PASSWORD pgcli -h $$POSTGRES_HOST -p $$POSTGRES_PORT -U $$POSTGRES_USER -d $$POSTGRES_DB
