APP_NAME=pr-reviewer

build:
	go build -o $(APP_NAME) ./cmd/server

run:
	go run ./cmd/server

run-local:
	docker-compose up -d db
	export DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" && \
	export APP_PORT="8080" && \
	make run

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

docker-restart:
	docker-compose restart app

test:
	./test_api.sh

.PHONY: build run run-local docker-up docker-down docker-logs docker-restart test
