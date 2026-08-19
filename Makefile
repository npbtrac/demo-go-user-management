.PHONY: test generate compose

test:
	go test ./...

generate:
	buf dep update
	buf generate

compose:
	docker compose -f deploy/docker/docker-compose.yml up --build
