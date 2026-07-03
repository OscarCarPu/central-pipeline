up:
	docker compose up -d --wait

down:
	docker compose down

restart:
	docker compose down -v
	docker compose up -d --build --wait

