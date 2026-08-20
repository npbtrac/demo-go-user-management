.PHONY: test generate dev compose compose-up compose-build compose-rebuild compose-down smoke smoke-public smoke-internal smoke-service

test:
	go test ./...

generate:
	buf dep update
	buf generate

# Host live-reload with Air. Start Postgres first: make compose-up-db
dev:
	go run github.com/air-verse/air@v1.61.7

compose: compose-up

compose-up:
	docker compose up -d

compose-up-db:
	docker compose up -d postgres

compose-build:
	docker compose build

compose-rebuild:
	docker compose up -d --build

compose-down:
	docker compose down

# Smoke-test the running local stack.
smoke:
	go run ./scripts/smoke

smoke-public:
	go run ./scripts/smoke -only=public

smoke-internal:
	go run ./scripts/smoke -only=internal

smoke-service:
	go run ./scripts/smoke -only=service
