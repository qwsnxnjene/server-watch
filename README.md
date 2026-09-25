# server-watch

`server-watch` — учебный проект на Go для мониторинга состояния Linux-системы.

Сервис собирает системные метрики, предоставляет их через HTTP API и сохраняет историю измерений в SQLite. Архитектура проекта постепенно расширяется в сторону production-like сервиса: добавляются кэширование, Redis, алерты, фоновые worker'ы, уведомления, конфигурация и observability.

## Что уже реализовано

На текущий момент проект включает:

* HTTP API для получения состояния системы;
* сбор метрик:

  * CPU;
  * RAM;
  * Disk;
* endpoint `/health` для проверки состояния сервиса;
* endpoint `/metrics` для получения текущих метрик;
* endpoint `/history` для получения исторических метрик;
* endpoint `/alerts` для получения алертов;
* Prometheus endpoint для сбора метрик;
* Визуализация метрик с Grafana;
* сохранение истории метрик в SQLite;
* получение истории метрик через API;
* graceful shutdown HTTP-сервера;
* корректную работу с `context.Context`;
* отмену фоновых операций при завершении приложения;
* Redis для кэширования метрик;
* Redis health check;
* Redis-backed состояние алертов с fallback на in-memory storage;
* систему алертов по CPU и памяти;
* очередь уведомлений;
* отправку уведомлений через HTTP;
* фонового notification worker;
* конфигурацию приложения;
* request-scoped логирование через `slog`;
* `request_id` для сквозной корреляции логов HTTP-запроса;
* тесты основных компонентов;
* проверку проекта с помощью `go test ./...`;
* проверку на race condition с помощью `go test -race ./...`.

Проект развивается поэтапно: каждый новый этап добавляет инфраструктурную или архитектурную возможности, не ломая уже существующую функциональность.

# Этап 5 — Request ID и сквозная корреляция логов

## Цель этапа

Добавить в HTTP-сервис сквозной идентификатор запроса `request_id`, который позволяет связать между собой все логи, относящиеся к одному HTTP-запросу.

Полноценный distributed tracing на базе OpenTelemetry на этом этапе не внедряется.

Основная задача этапа — реализовать простую и прозрачную корреляцию запросов через `request_id`.

---

## Что необходимо реализовать

Для каждого входящего HTTP-запроса:

* если клиент передал заголовок `X-Request-ID` — использовать его;
* если заголовок отсутствует — сгенерировать новый UUID;
* добавить `request_id` в `context.Context`;
* вернуть `X-Request-ID` в HTTP-ответе;
* создать request-scoped logger с `request_id`;
* передавать logger через `context.Context` в нижние слои приложения;
* использовать этот logger во всех местах, где выполняется логирование в рамках HTTP-запроса.

### Логирование

Для request-scoped логирования используется:

```go
slog.With("request_id", requestID)
```

Полученный logger сохраняется в context и может быть использован в service/repository слоях.

---

# Архитектура

После добавления request ID обработка HTTP-запроса выглядит следующим образом:

```text
HTTP request
      │
      ▼
RequestIDMiddleware
      │
      ├── X-Request-ID получен
      │        или
      └── UUID сгенерирован
      │
      ▼
context.Context
      │
      ├── request_id
      │
      └── request-scoped logger
      │
      ▼
HTTP Handler
      │
      ▼
System
      │
      ▼
Repository / Cache / Redis
```

Request ID при этом не создаётся заново на каждом слое.

Он создаётся один раз на границе HTTP и затем передаётся вместе с context.

---

# Request ID Middleware

Middleware выполняет несколько задач:

1. Получает `X-Request-ID` из HTTP-запроса.
2. Если ID отсутствует — генерирует UUID.
3. Добавляет ID в context.
4. Добавляет ID в HTTP-ответ.
5. Передаёт запрос следующему handler.

Концептуально:

```text
request
   │
   ├── X-Request-ID
   │
   ▼
RequestIDMiddleware
   │
   ├── request_id → context
   │
   └── X-Request-ID → response
   │
   ▼
next handler
```

---

# Context

Для `request_id` используется отдельный внутренний ключ:

```go
type contextKey string

const requestIDKey contextKey = "request_id"
```

Context используется для передачи request-scoped данных между слоями.

Важно сохранять исходный context запроса:

```go
r.Context()
```

а не создавать новый context через `context.Background()`.

Это позволяет сохранить cancellation и deadline исходного HTTP-запроса.

---

# Request-scoped Logger

После получения `request_id` handler создаёт logger:

```go
logger := slog.With("request_id", requestID)
```

После этого logger сохраняется в context через пакет `internal/logger`.

Публичный API пакета предоставляет функции:

```go
WithLogger(ctx, logger)
FromContext(ctx)
```

Сам ключ `loggerKey` остаётся скрытой деталью реализации пакета.

Это позволяет другим слоям получать logger, не зная, каким именно образом он хранится в context.

---

# Передача logger между слоями

В HTTP handler:

```text
request
   │
   ▼
request_id
   │
   ▼
request-scoped logger
   │
   ▼
context
   │
   ▼
System
   │
   ▼
Repository / Redis / другие компоненты
```

Например, service может получить logger:

```go
logger, ok := logger.FromContext(ctx)
if !ok {
    logger = slog.Default()
}
```

После этого логирование автоматически содержит `request_id`.

---

# Где используется request-scoped logger

Logger используется только там, где действительно выполняется логирование.

Например:

```go
logger.Warn(
    "не удалось получить метрики из кэша",
    "error", err,
)
```

Если logger был создан для HTTP-запроса, запись автоматически содержит:

```text
request_id
```

Дополнительный `request_id` в каждом нижнем слое извлекать не требуется.

---

# Где request_id не используется

Не все операции приложения являются частью HTTP-запроса.

Например, фоновый сбор метрик:

```text
background collector
      │
      ▼
CollectMetrics
      │
      ▼
updateAlerts
```

не имеет HTTP request ID.

Поэтому фоновые операции не получают искусственный `request_id`.

Они используют обычный application logger:

```go
slog.Warn(...)
```

если им требуется логирование.

Это важно, поскольку `request_id` идентифицирует именно конкретный входящий HTTP-запрос.

---

# Логирование ошибок

В проекте используется принцип:

> Ошибка по возможности передаётся выше, а логируется на подходящей границе обработки.

Например:

```text
Repository
    │
    └── return error
          │
          ▼
System
    │
    └── return error
          │
          ▼
Handler
    │
    └── logger.Error(...)
```

Это позволяет избежать многократного логирования одной и той же ошибки на разных уровнях.

Исключение — фоновые операции или ошибки, которые не передаются выше и должны быть обработаны непосредственно в месте возникновения.

---

# Пример итогового лога

При запросе:

```text
GET /metrics
```

сервер может записать:

```json
{
  "level": "INFO",
  "msg": "получен запрос",
  "request_id": "fb50df12-60c5-4013-ac5f-f38dd2a70021",
  "path": "/prometheus"
}
```

Другие логи в рамках того же HTTP-запроса используют тот же `request_id`.

Таким образом, по одному идентификатору можно найти все связанные события.

---

# Тестирование

Проверяется несколько сценариев.

## Клиент передал X-Request-ID

Например:

```text
X-Request-ID: test-123
```

Ожидается:

* context содержит `test-123`;
* response содержит `X-Request-ID: test-123`;
* logger использует `test-123`.

---

## Клиент не передал X-Request-ID

Ожидается:

* генерируется новый UUID;
* UUID попадает в context;
* тот же UUID возвращается в `X-Request-ID`;
* logger использует тот же ID.

Корректность UUID проверяется через UUID parser.

---

## Logger propagation

Проверяется, что request-scoped logger может быть передан из HTTP handler в service layer через context.

---

## Общие проверки

После изменений должны проходить:

```bash
go test ./...
```

и:

```bash
go test -race ./...
```

Race detector не должен обнаруживать race condition.

---

# Что сознательно не входит в этап

На этом этапе не внедряется полноценный OpenTelemetry tracing.

То есть не добавляются:

* `TracerProvider`;
* spans;
* trace ID;
* OTLP exporter;
* Jaeger;
* distributed tracing;
* автоматическая instrumentация HTTP/Redis/SQL.

Сквозная корреляция ограничена `request_id`.

---

# Результат этапа

После завершения этапа приложение умеет:

* идентифицировать каждый HTTP-запрос;
* сохранять идентификатор запроса в context;
* возвращать идентификатор клиенту;
* создавать request-scoped logger;
* передавать logger через context между слоями;
* связывать логи одного HTTP-запроса через `request_id`;
* корректно отделять HTTP-запросы от фоновых операций;
* не дублировать логирование одной и той же ошибки на разных слоях.

Таким образом, базовая сквозная корреляция HTTP-запросов реализована без внедрения полноценной системы distributed tracing.
