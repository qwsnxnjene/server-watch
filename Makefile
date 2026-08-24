run:
	go run ./cmd/server-watch/main.go

vet:
	go vet ./...

lint:
	golangci-lint run

check:
	go vet ./...
	golangci-lint run