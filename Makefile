up:
	docker compose up -d --wait db

down:
	docker compose down

restart:
	docker compose down -v
	docker compose up -d --build --wait db

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

dbt-deps:
	docker compose run --rm dbt deps

dbt-run:
	docker compose run --rm dbt build

consumer-up:
	docker compose up -d --build consumer

consumer-logs:
	docker compose logs -f consumer
