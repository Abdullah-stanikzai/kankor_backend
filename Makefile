# Makefile for Kankor Backend

.PHONY: build run dev test clean deps docker-up docker-down docker-logs

# Build the application
build:
	go build -o bin/kankor-backend .

# Run the application
run: build
	./bin/kankor-backend

# Run in development mode with hot reload
dev:
	reflex -r '\.go$$' -s -- sh -c 'go run main.go'

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod tidy
	go mod download

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-rebuild:
	docker-compose up -d --build

# Database commands
db-migrate:
	psql -h localhost -U postgres -d kankor_db -f db_schema.sql

db-reset:
	docker-compose down -v
	docker-compose up -d postgres
	sleep 5
	psql -h localhost -U postgres -d kankor_db -f db_schema.sql

# Health check
health:
	curl -f http://localhost:8080/health || echo "Health check failed"

# API testing examples
test-auth:
	curl -X POST http://localhost:8080/api/v1/auth/send-otp \
		-H "Content-Type: application/json" \
		-d '{"phone_number": "93700123456", "purpose": "login"}'

test-health:
	curl http://localhost:8080/health