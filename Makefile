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

swagger:
	go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go -o docs --parseInternal

format:
	go fmt ./...

lint:
	go vet ./...
