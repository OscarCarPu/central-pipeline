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
