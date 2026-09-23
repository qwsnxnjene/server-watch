package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"server-watch/internal/config"
	"server-watch/internal/handlers"
	"server-watch/internal/notifications"
	redis2 "server-watch/internal/redis"
	"server-watch/internal/storage"
	"server-watch/internal/system"
	"server-watch/internal/system/alert-state-store"
	"sync"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// отменяем общий контекст при SIGINT/SIGTERM,
	// чтобы остановить фоновые горутины и завершить сервис корректно
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, repo, err := setupDatabase()
	if err != nil {
		slog.Error("ошибка инициализации базы данных", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("не удалось загрузить конфигурацию", "error", err)
		os.Exit(1)
	}
	slog.Info("конфигурация загружена",
		"cpu_threshold", cfg.CPUThreshold,
		"mem_threshold", cfg.MemThreshold,
		"trigger_count", cfg.TriggerCount,
		"resolve_count", cfg.ResolveCount,
		"slack_enabled", cfg.SlackEnabled,
	)

	client := redis2.NewClient()
	cache := redis2.NewRedisMetricsCache(client, 30*time.Second, "server-watch:metrics:")
	go redis2.StartRedisHealthCheck(ctx, client, 10*time.Second)

	// Redis используется как основное хранилище состояния алертов.
	// При недоступности Redis сервис переключается на in-memory fallback
	redisAlertState := redis2.NewRedisAlertStateStore(
		client,
		time.Minute,
		"server-watch:alert:",
	)
	memoryAlertState := alert_state_store.NewInMemoryAlertStateStore()
	alertStateStore := system.NewFallbackAlertStateStore(
		memoryAlertState,
		redisAlertState,
	)
	notificationQueue := redis2.NewRedisNotificationQueue(client, "server-watch:")

	var senders []notifications.Sender

	if cfg.SlackEnabled {
		slackSender := notifications.NewHttpSender(http.DefaultClient, cfg.SlackURL)
		senders = append(senders, slackSender)
	}
	worker := notifications.NewWorker(notificationQueue, senders)

	// Запускаем worker, который читает уведомления из очереди
	// и передаёт их зарегистрированным отправителям
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Run(ctx)
	}()

	sys := system.NewSystem(repo, alertStateStore, cfg, "config.yaml", cache, notificationQueue)
	err = sys.CollectMetrics(ctx)
	if err != nil {
		slog.Error("не удалось прочитать метрики при запуске", "error", err)
		os.Exit(1)
	}

	if err = system.RegisterPrometheusMetrics(); err != nil {
		slog.Error("не удалось зарегистрировать метрики Prometheus", "error", err)
		os.Exit(1)
	}
	promHandler := system.PrometheusHandler()

	metricsErrCh := startMetricsCollector(ctx, sys, &wg)

	server := newHTTPServer(sys, &promHandler)
	serverErrCh := startHTTPServer(server, &wg)

	select {
	case <-ctx.Done():
		slog.Info("получен сигнал остановки, начинаем процесс graceful shutdown")
	case err = <-metricsErrCh:
		slog.Error("сбор метрик завершился с ошибкой", "error", err)
		cancel()
	case err = <-serverErrCh:
		slog.Error("ошибка сервера", "error", err)
		cancel()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("ошибка при остановке сервера", "error", err)
		os.Exit(1)
	}

	wg.Wait()
	slog.Info("выполнение программы остановлено")
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

// startMetricsCollector запускает периодический сбор системных метрик
// и возвращает канал для передачи критической ошибки сборщика
func startMetricsCollector(ctx context.Context, sys *system.System, wg *sync.WaitGroup) <-chan error {
	errCh := make(chan error, 1)

	wg.Add(1)

	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		defer wg.Done()
		defer close(errCh)

		for {
			select {
			case <-ticker.C:
				err := sys.CollectMetrics(ctx)
				if err != nil {
					errCh <- err
					return
				}
			case <-ctx.Done():
				slog.Info("останавливаем обновление метрик")
				return
			}
		}
	}()

	return errCh
}

func startHTTPServer(server *http.Server, wg *sync.WaitGroup) <-chan error {
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()

		slog.Info("сервер запущен на localhost:8080")
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("сервер корректно завершил свою работу")
			} else {
				errCh <- err
				return
			}
		}
	}()

	return errCh
}

// setupDatabase открывает SQLite-базу, применяет миграции
// и создаёт репозиторий для работы с ней
func setupDatabase() (*sql.DB, *storage.SQLiteRepository, error) {
	db, err := storage.NewSQLite("server-watch.db")
	if err != nil {
		return nil, nil, err
	}

	if err := storage.Migrate(db); err != nil {
		db.Close()
		return nil, nil, err
	}
	repo := storage.NewSQLiteRepository(db)

	return db, repo, nil
}
