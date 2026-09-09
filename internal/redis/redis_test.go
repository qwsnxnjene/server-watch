package redis

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	redisClient, err := NewClient()
	if err != nil {
		t.Fatalf("не удалось подключиться к Redis: %v", err)
	}

	if redisClient == nil {
		t.Fatalf("получен пустой клиент Redis")
	}
}
