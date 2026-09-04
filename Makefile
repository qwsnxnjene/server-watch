run:
	go run ./cmd/server-watch/main.go

up:
	docker compose up -d
	go run ./cmd/server-watch/main.go

stop:
	docker compose down

check:
	go vet ./...
	golangci-lint run

test:
	go test ./... -v