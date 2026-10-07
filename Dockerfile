FROM golang:1.27.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" \
    -o server-watch \
    ./cmd/server-watch

FROM scratch

WORKDIR /app

COPY --from=builder /app/server-watch .
COPY config.yaml .

EXPOSE 8080

ENTRYPOINT ["./server-watch"]