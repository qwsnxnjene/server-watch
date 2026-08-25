package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"server-watch/internal/handlers"
	"server-watch/internal/storage"
	"server-watch/internal/system"
	"sync"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	wg := sync.WaitGroup{}

	db, err := storage.NewSQLite("server-watch.db")
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}
	defer db.Close()

	if err := storage.Migrate(db); err != nil {
		log.Fatalf("[ERROR] %v", err)
	}

	sys := system.NewSystem()
	err = sys.CollectMetrics()
	if err != nil {
		log.Fatalf("[ERROR] не удалось прочитать метрики при запуске: %v", err)
	}

	wg.Add(1)
	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		defer wg.Done()

		for {
			select {
			case <-ticker.C:
				err := sys.CollectMetrics()
				if err != nil {
					log.Printf("[ERROR] не удалось прочитать метрики: %v", err)
				}
			case <-ctx.Done():
				log.Printf("[INFO] останавливаем обновление метрик")
				return
			}
		}
	}(ctx)

	server := newHTTPServer(sys)

	go func() {
		log.Printf("[INFO] сервер запущен на localhost:8080")
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Print("[INFO] сервер корректно завершил свою работу")
			} else {
				log.Printf("[ERROR] сервер завершил работу: %v", err)
			}
		}
	}()

	<-ctx.Done()
	log.Printf("[INFO] получен сигнал остановки, начинаем процесс graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[ERROR] ошибка при остановке сервера: %v", err)
	}
	log.Println("[INFO] выполнение программы остановлено")
	wg.Wait()
}

func newHTTPServer(sys *system.System) *http.Server {
	handler := handlers.NewHandler(sys)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", handler.MetricsHandler)
	mux.HandleFunc("/health", handler.HealthHandler)

	return &http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}
}
