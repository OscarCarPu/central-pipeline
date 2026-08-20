# Colors
RED=\033[0;31m
GREEN=\033[0;32m
CYAN=\033[0;36m
NC=\033[0m

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

# All tests: silent, only prints pass/fail. Run by the pre-commit hook.
test-silent:
	@printf "$(CYAN)>>> Starting database...$(NC)\n"
	@docker compose up -d --wait db > /dev/null 2>&1 || { printf "$(RED)>>> Database failed to start$(NC)\n"; exit 1; }
	@printf "$(CYAN)>>> Running all tests...$(NC)\n"
	@go test ./... > /dev/null 2>&1 || { printf "$(RED)>>> Unit tests failed$(NC)\n"; exit 1; }
	@set -a; . ./.env; set +a; go test -tags=integration ./... > /dev/null 2>&1 || { printf "$(RED)>>> Integration tests failed$(NC)\n"; exit 1; }
	@docker compose run --rm dbt build > /dev/null 2>&1 || { printf "$(RED)>>> dbt build failed$(NC)\n"; exit 1; }
	@printf "$(GREEN)>>> All tests passed$(NC)\n"

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
