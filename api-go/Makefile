.PHONY: build build-api build-migrate build-worker run-api run-migrate run-worker test fmt vet tidy clean sqlc

build: build-api build-migrate build-worker

# Requires sqlc >= 1.29 on PATH (Windows: download release zip; do not use 1.28).
sqlc:
	sqlc generate

build-api:
	go build -o bin/api.exe ./cmd/api

build-migrate:
	go build -o bin/migrate.exe ./cmd/migrate

build-worker:
	go build -o bin/worker.exe ./cmd/worker

run-api:
	go run ./cmd/api

run-migrate:
	go run ./cmd/migrate

run-worker:
	go run ./cmd/worker

test:
	go test ./...

fmt:
	gofmt -l -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin
