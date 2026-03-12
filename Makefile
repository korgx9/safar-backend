APP_NAME=safar-api

.PHONY: run build test test-verbose docker-up docker-down deps tidy swagger format lint clean

run:
	go run cmd/api/main.go

build:
	mkdir -p bin
	go build -o bin/$(APP_NAME) cmd/api/main.go

test:
	go test ./...

test-verbose:
	go test -v ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

deps:
	go mod tidy

tidy: deps

swagger:
	go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go -o docs --parseInternal

format:
	go fmt ./...

lint:
	go vet ./...

clean:
	rm -rf bin
	go clean -testcache
