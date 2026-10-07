run:
	go run ./cmd/server-watch/main.go

up:
	docker compose up -d

stop:
	docker compose down

check:
	go vet ./...
	golangci-lint run

test:
	go test ./... -v

benchmark:
	go test -bench=. -benchmem ./...