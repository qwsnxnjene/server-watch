# server-watch

Сервис мониторинга состояния Linux-систем на Go, который собирает метрики CPU/RAM/Disk, отдаёт их через HTTP API и Prometheus, хранит историю в SQLite, умеет алертить по порогам с уведомлениями в Slack и разворачивается в Docker/Kubernetes.

Проект сделан поэтапно от простого скрипта CLI до сервиса с Redis-кэшем, фоновыми воркерами, observability и Helm-чартом. История всех этапов — в [`tasks/`](./tasks).

## Содержание

- [Возможности](#возможности)
- [Архитектура](#архитектура)
- [Технологии](#технологии)
- [Быстрый старт](#быстрый-старт)
  - [Локально](#локально)
  - [Docker Compose](#docker-compose)
  - [Kubernetes (Helm)](#kubernetes-helm)
- [HTTP API](#http-api)
- [Конфигурация](#конфигурация)
- [Тесты и бенчмарки](#тесты-и-бенчмарки)
- [Структура проекта](#структура-проекта)
- [Известные ограничения](#известные-ограничения)

## Возможности

**Сбор и выдача метрик**
- Сбор CPU, RAM и Disk каждые 5 секунд в фоновой горутине с кэшированием в памяти и в Redis.
- `/metrics` отдаёт текущие значения в JSON или в формате Prometheus - в зависимости от заголовка `Accept`.
- История измерений сохраняется в SQLite и доступна через `/history` с фильтром по периоду.

**Алерты**
- Пороговые алерты по CPU и памяти: отдельные счётчики срабатывания (`trigger_count`) и восстановления (`resolve_count`), чтобы не алертить на единичный всплеск.
- Состояние алертов хранится в Redis (атомарная запись через embedded Lua-скрипт) с автоматическим fallback на in-memory хранилище при недоступности Redis и синхронизацией состояния после его восстановления.
- Уведомления уходят через очередь в Redis и фоновый worker, с отправкой в Slack (включается конфигурацией).

**Observability**
- Структурированное логирование через `log/slog` в JSON.
- Сквозная корреляция логов одного HTTP-запроса через `request_id` (из заголовка `X-Request-ID` или сгенерированный UUID).
- Метрики Prometheus, включая гистограмму `http_request_duration_seconds` по каждому эндпоинту (RPS, P50/P95/P99), и готовые дашборды Grafana.
- Опциональное профилирование через `net/http/pprof`, включается конфигурацией.

**Эксплуатация**
- Graceful shutdown по SIGINT/SIGTERM: сервер и фоновые горутины останавливаются согласованно через `context.Context` и `sync.WaitGroup`.
- Конфигурация загружается из YAML, переопределяется переменными окружения и валидируется при старте; часть параметров можно менять на лету через `/config`.
- Docker-образ на базе `scratch` (multi-stage build, статический бинарник), Docker Compose со всем стеком (Redis, Prometheus, Grafana), Helm-чарт и сырые манифесты Kubernetes.

**Качество кода**
- 78 тестов (~4000 строк, больше, чем самого кода) и 10 бенчмарков `testing.B`, включая интеграционные тесты Redis и SQLite.
- Проверка на race condition (`go test -race ./...`), `go vet` и `golangci-lint` в `make check`.
- Нагрузочное тестирование HTTP API задокументировано в [`docs/benchmarks.md`](./docs/benchmarks.md), включая разбор и устранение `SQLITE_BUSY` под нагрузкой через WAL-режим.

## Архитектура

```text
                     HTTP-запрос
                          │
                          ▼
              RequestIDMiddleware (request_id)
                          │
                          ▼
              MetricsMiddleware (latency → Prometheus)
                          │
                          ▼
                     HTTP Handler
                          │
            ┌─────────────┼──────────────┐
            ▼             ▼              ▼
         System      Repository       Redis
      (бизнес-логика)  (SQLite)    (cache, alert state,
            │                        notification queue)
            ▼
      фоновый сборщик метрик (ticker, 5s)
            │
            ▼
       обновление алертов → очередь уведомлений → notification worker → Slack
```

Инфраструктура в Docker Compose:

```text
              ┌──────────────┐
              │    Grafana   │
              │    :3000     │
              └──────┬───────┘
                     ▼
              ┌──────────────┐
              │  Prometheus  │
              │    :9090     │
              └──────┬───────┘
                     │ scrape
                     ▼
┌──────────────┐  ┌──────────────┐
│    Redis     │◄─│ server-watch │
│    :6379     │  │    :8080     │
└──────────────┘  └──────┬───────┘
                         ▼
                  SQLite (volume)
```

## Технологии

| Область | Используется |
|---|---|
| Язык | Go 1.27 |
| HTTP | стандартный `net/http` (`http.ServeMux`) |
| Хранилище истории | SQLite (`modernc.org/sqlite`, без cgo) |
| Кэш и состояние | Redis (`redis/go-redis/v9`) |
| Метрики | `prometheus/client_golang` + Grafana |
| Логирование | `log/slog`, JSON-формат |
| Конфигурация | YAML (`gopkg.in/yaml.v3`) + переменные окружения |
| Контейнеризация | Docker (multi-stage, `scratch`), Docker Compose |
| Оркестрация | Kubernetes (manifests + Helm chart) |
| Качество | `go vet`, `golangci-lint`, `go test -race`, бенчмарки |

## Быстрый старт

### Локально

Нужен установленный Go и Redis (слушающий `localhost:6379`).

```bash
git clone https://github.com/qwsnxnjene/server-watch.git
cd server-watch
make run
# или напрямую:
go run ./cmd/server-watch
```

При первом запуске создастся `config.yaml` со значениями по умолчанию и файл `server-watch.db` (SQLite). Сервис поднимется на `localhost:8080`.

### Docker Compose

Поднимает сразу весь стек: сам сервис, Redis, Prometheus и Grafana.

```bash
make up
# или:
docker compose up -d
```

- API: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (datasource на Prometheus уже прописан)

SQLite и `config.yaml` вынесены из образа через volume/bind-mount, так что `docker compose down && docker compose up -d` не теряет историю метрик.

```bash
make stop
# или:
docker compose down
```

### Kubernetes (Helm)

```bash
helm lint helm/server-watch
helm template server-watch helm/server-watch

helm install server-watch helm/server-watch
helm upgrade server-watch helm/server-watch
helm rollback server-watch 1
```

Chart создаёт `Deployment`, `Service` (ClusterIP, порт 8080), `ConfigMap` с начальной конфигурацией, `Secret` для `SLACK_URL` и два `PersistentVolumeClaim` (для SQLite и для изменяемого `config.yaml`). Сервис наружу не публикуется — для проверки изнутри кластера:

```bash
kubectl run curl --rm -it --image=curlimages/curl --restart=Never \
  -- curl http://server-watch:8080/health
```

Сырые манифесты без Helm лежат в [`k8s/`](./k8s).

## HTTP API

| Метод | Путь | Описание |
|---|---|---|
| GET | `/health` | Статус сервиса: `200` если метрики читаются и обновлялись не дольше 10с назад, иначе `503` |
| GET | `/metrics` | Текущие метрики в JSON; при `Accept: text/plain` или `application/openmetrics-text` — формат Prometheus |
| GET | `/history?from=&to=` | История метрик за период (`RFC3339`); по умолчанию — последний час |
| GET | `/alerts?active_only=` | Список алертов, `active_only=true` — только активные |
| POST | `/config` | Частичное обновление конфигурации; возвращает итоговый конфиг |
| GET | `/debug/pprof/*` | Профилирование pprof, только если `PPROF_ENABLED=true` |

<details>
<summary>Примеры запросов и ответов</summary>

`GET /health`

```json
{
  "status": "ok",
  "last_collection": "2026-10-07T18:40:12Z"
}
```

`GET /metrics`

```json
{
  "cpu_percent": 12.5,
  "mem_percent": 45.2,
  "mem_used_mb": 3840,
  "mem_total_mb": 8192,
  "disk_percent": 60.1,
  "disk_used_gb": 45,
  "disk_total_gb": 75
}
```

`GET /alerts?active_only=true`

```json
{
  "alerts": [
    {
      "id": 14,
      "alert_type": "high_cpu",
      "ts": "2026-10-07T18:39:00Z",
      "threshold": 80,
      "resolved": false,
      "resolvedAt": null,
      "value": 91.4
    }
  ]
}
```

`POST /config` — тело запроса (только изменяемые поля):

```json
{
  "cpu_threshold": 85,
  "slack_enabled": true,
  "slack_url": "https://hooks.slack.com/services/..."
}
```

</details>

## Конфигурация

Порядок применения: значения по умолчанию → `config.yaml` → переменные окружения → валидация. При первом запуске, если файла конфигурации нет, он создаётся автоматически (`0600`).

| Параметр YAML | Переменная окружения | По умолчанию | Описание |
|---|---|---|---|
| `cpu_threshold` | `CPU_THRESHOLD` | `80` | Порог CPU, % |
| `mem_threshold` | `MEM_THRESHOLD` | `90` | Порог памяти, % |
| `trigger_count` | `TRIGGER_COUNT` | `3` | Сколько подряд превышений порога нужно для алерта |
| `resolve_count` | `RESOLVE_COUNT` | `3` | Сколько подряд значений в норме нужно, чтобы снять алерт |
| `slack_enabled` | `SLACK_ENABLED` | `false` | Включить отправку уведомлений в Slack |
| `slack_url` | `SLACK_URL` | `""` | Webhook URL (обязателен http/https при `slack_enabled: true`) |
| `pprof_enabled` | `PPROF_ENABLED` | `false` | Включить `/debug/pprof/*` |
| — | `CONFIG_PATH` | `config.yaml` | Путь к файлу конфигурации |
| — | `DB_PATH` | `server-watch.db` | Путь к файлу SQLite |
| — | `REDIS_ADDR` | `localhost:6379` | Адрес Redis |

Полный пример — в [`.env.example`](./.env.example).

## Тесты и бенчмарки

```bash
make test        # go test ./... -v
make check        # go vet ./... + golangci-lint run
make benchmark    # go test -bench=. -benchmem ./...
```

Для тестов пакета `internal/redis` нужен запущенный Redis (`docker compose up -d redis` или локальный на `6379`).

Race-детектор:

```bash
go test -race ./...
```

Нагрузочное тестирование HTTP API (`hey`, 10 000 запросов, concurrency=200) и подробный разбор найденной под нагрузкой проблемы `SQLITE_BUSY` и её решения через `PRAGMA journal_mode=WAL` — в [`docs/benchmarks.md`](./docs/benchmarks.md). Коротко: `/health` держит ~34k RPS, `/history` (самый тяжёлый эндпоинт, чтение из SQLite) - около 4.9k RPS после перехода на WAL, с P95 ~ 93 мс.

## Структура проекта

```text
cmd/server-watch/        точка входа, сборка зависимостей, HTTP-сервер
internal/
  config/                загрузка/валидация/сохранение конфигурации
  handlers/               HTTP-хендлеры и middleware (request-id, метрики)
  logger/                 request-scoped логгер поверх context.Context
  notifications/          очередь, воркер, отправители уведомлений (Slack)
  prometheus/             регистрация и хендлер метрик Prometheus
  redis/                  клиент Redis: кэш метрик, состояние алертов, очередь
  storage/                SQLite-репозиторий и миграции
  system/                 бизнес-логика: сбор метрик, алерты, конфигурация
    alert-state-store/    in-memory реализация хранилища состояния алертов
    model/                доменные модели (Metrics, Alert, AlertState)
docs/benchmarks.md        методика и результаты нагрузочного тестирования
k8s/                       сырые манифесты Kubernetes
helm/server-watch/         Helm-чарт
grafana/provisioning/      автопровижининг datasource для Grafana
tasks/                      ТЗ и критерии приёмки по каждому этапу разработки
```

## Известные ограничения

Осознанные упрощения, оставленные вне текущего объёма проекта:

- Одна реплика; SQLite не рассчитан на несколько одновременно пишущих экземпляров сервиса.
- Docker-образ на `scratch` - около 32 МБ, внутри нет шелла и диагностических утилит (отладка через `/health`, `docker logs`/`kubectl logs`, pprof).
- Контейнер запускается от root.
- Prometheus и Grafana поднимаются через Docker Compose, но не входят в Helm-чарт; `ServiceMonitor` не реализован.
- Нет CI/CD pipeline, Ingress, TLS и service mesh.

Подробности по каждому этапу разработки в [`tasks/`](./tasks).
