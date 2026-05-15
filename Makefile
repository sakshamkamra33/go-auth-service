.PHONY: run test build lint clean docker-up docker-down

## run: Start the development server
run:
	JWT_SECRET=dev-secret-32-bytes-long-pad-ok! go run ./cmd/server

## build: Compile to binary
build:
	go build -o secure-auth ./cmd/server

## test: Run all unit tests
test:
	go test -v ./...

## test-cover: Run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

## lint: Run static analysis (requires golangci-lint)
lint:
	golangci-lint run ./...

## tidy: Clean up go.mod and go.sum
tidy:
	go mod tidy

## docker-up: Start with Docker Compose
docker-up:
	JWT_SECRET=dev-secret-32-bytes-long-pad-ok! docker compose up --build

## docker-down: Stop Docker containers
docker-down:
	docker compose down

## clean: Remove build artifacts
clean:
	rm -f secure-auth coverage.out

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //'
