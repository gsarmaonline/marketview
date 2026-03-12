.PHONY: up down rebuild test

up:
	docker compose up -d

down:
	docker compose down

rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d

test:
	go test ./...
