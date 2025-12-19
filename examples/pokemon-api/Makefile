.PHONY: help build run test clean docker-build docker-run

help:
	@echo "Pokemon API - Makefile targets:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application"
	@echo "  make test         - Run all tests"
	@echo "  make test-api     - Run API integration tests"
	@echo "  make test-e2e     - Run E2E tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run with docker-compose"

build:
	go build -o pokemon-api cmd/main.go

run: build
	./pokemon-api

test:
	go test -v ./tests/...

test-api:
	go test -v ./tests/api_test.go

test-e2e:
	go test -v ./tests/e2e_test.go

clean:
	rm -f pokemon-api
	rm -f internal/db/pokemon.db*

docker-build:
	docker build -t pokemon-api .

docker-run:
	docker-compose up -d

docker-logs:
	docker-compose logs -f

docker-stop:
	docker-compose down
