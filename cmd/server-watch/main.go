package main

import (
	"context"
	"errors"
	"log/slog"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, err := storage.NewSQLite("server-watch.db")
	if err != nil {
		slog.Error("не удалось создать/открыть SQLite базу данных", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := storage.Migrate(db); err != nil {
		slog.Error("не удалось провести миграцию", "error", err)
		os.Exit(1)
	}
	repo := storage.NewSQLiteRepository(db)

	sys := system.NewSystem(repo)
	err = sys.CollectMetrics()
	if err != nil {
		slog.Error("не удалось прочитать метрики при запуске", "error", err)
		os.Exit(1)
	}

	if err = system.RegisterPrometheusMetrics(); err != nil {
		slog.Error("не удалось зарегистрировать метрики Prometheus", "error", err)
		os.Exit(1)
	}
	promHandler := system.PrometheusHandler()

	wg := sync.WaitGroup{}
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
					slog.Error("не удалось прочитать метрики", "error", err)
					os.Exit(1)
				}
			case <-ctx.Done():
				slog.Info("останавливаем обновление метрик")
				return
			}
		}
	}(ctx)

	server := newHTTPServer(sys, &promHandler)

	go func() {
		slog.Info("сервер запущен на localhost:8080")
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("сервер корректно завершил свою работу")
			} else {
				slog.Error("сервер завершил работу", "error", err)
				os.Exit(1)
			}
		}
	}()

	<-ctx.Done()
	slog.Info("получен сигнал остановки, начинаем процесс graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("ошибка при остановке сервера", "error", err)
		os.Exit(1)
	}
	slog.Info("выполнение программы остановлено")
	wg.Wait()
}

func newHTTPServer(sys *system.System, promHandler *http.Handler) *http.Server {
	handler := handlers.NewHandler(sys, *promHandler)

	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", handler.MetricsHandler)
	mux.HandleFunc("/health", handler.HealthHandler)
	mux.HandleFunc("/history", handler.HistoryHandler)
	mux.HandleFunc("/alerts", handler.AlertsHandler)

	return &http.Server{
		Addr:    "localhost:8080",
		Handler: mux,
	}
}
