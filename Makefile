.PHONY: up down rebuild test screenshot

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

# Capture screenshots of all frontend pages.
# Starts docker-compose if not already running; set STOP_SERVER=1 to stop after.
screenshot:
	cd frontend && npm run screenshot
