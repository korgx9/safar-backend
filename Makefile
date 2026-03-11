APP_NAME=safar-api

run:
	go run cmd/api/main.go

build:
	go build -o bin/$(APP_NAME) cmd/api/main.go

test:
	go test ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

deps:
	go mod tidy

format:
	go fmt ./...

lint:
	go vet ./...
