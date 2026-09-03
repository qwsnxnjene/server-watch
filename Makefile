run:
	go run ./cmd/server-watch/main.go

check:
	go vet ./...
	golangci-lint run

test:
	go test ./... -v