.PHONY: build test test-integration up down logs migrate

build:
	go build ./cmd/api ./cmd/migrate

test:
	go test -race ./...

test-integration:
	test -n "$${TEST_DATABASE_URL}" || (echo "TEST_DATABASE_URL is required" && exit 1)
	TEST_DATABASE_URL="$${TEST_DATABASE_URL}" go test -race -run Integration ./internal/repository/postgres

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f app

migrate:
	docker compose run --rm migrate
