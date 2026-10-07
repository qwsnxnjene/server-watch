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
* endpoint `/config` для изменения конфигурации приложения;
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
* проверку на race condition с помощью `go test -race ./...`;
* бенчмарк-тесты критических функций;
* middleware для измерения производительности хэндлеров;
* сбор метрик для подсчета p50, p95, p99 и rps для каждого эндпоинта;
* возможность включить/выключить профилирование с помощью pprof.

Проект развивается поэтапно: каждый новый этап добавляет инфраструктурную или архитектурную возможности, не ломая уже существующую функциональность.

# Этап 7 — Docker и Kubernetes

## Цель этапа

Подготовить `server-watch` к запуску в контейнере и Kubernetes.

На этом этапе проект переводится от локального запуска приложения к воспроизводимому инфраструктурному окружению:

* собирается Docker image;
* используется multi-stage Docker build;
* приложение запускается в минимальном runtime image;
* внешние зависимости подключаются через Docker Compose;
* конфигурация и данные сохраняются вне контейнера;
* Kubernetes-манифесты объединяются в Helm chart;
* проверяется жизненный цикл приложения в Kubernetes;
* проверяется сохранение данных при пересоздании Pod.

Этап является учебным и не ставит целью полностью production-ready Kubernetes deployment.

---

## Docker

Для сборки приложения используется multi-stage Dockerfile.

На этапе сборки используется:

```text
golang:1.27.1
```

После компиляции бинарник переносится в минимальный runtime image:

```text
scratch
```

Приложение собирается как статический Linux binary:

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-s -w" \
    -o server-watch \
    ./cmd/server-watch
```

Runtime image содержит только:

* бинарник `server-watch`;
* `config.yaml`.

Проверка бинарника показала, что он статический, поэтому использование `scratch` возможно без дополнительной runtime-библиотеки.

---

## Docker image

После сборки размер image составляет примерно:

```text
32 MB
```

Формальный ориентир `<30 MB` в текущей реализации не достигнут.

При этом image уже значительно меньше типичного runtime image с полноценной Linux userland и содержит только необходимые для запуска приложения файлы.

---

## Особенность scratch image

Так как runtime использует:

```text
scratch
```

в контейнере отсутствуют:

* shell;
* `cat`;
* `sh`;
* `printenv`;
* другие стандартные Linux utilities.

Поэтому диагностика контейнера выполняется преимущественно через:

* HTTP endpoints приложения;
* `docker logs`;
* `kubectl logs`;
* внешние debug/curl-контейнеры.

---

## Docker Compose

Для локального инфраструктурного окружения используется:

```text
docker-compose.yml
```

Compose включает:

```text
server-watch
redis
prometheus
grafana
```

Схема окружения:

```text
              ┌──────────────┐
              │    Grafana   │
              │    :3000     │
              └──────┬───────┘
                     │
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
                         │
                         ▼
                  SQLite volume
```

`server-watch` получает адрес Redis через:

```text
REDIS_ADDR=redis:6379
```

Prometheus получает метрики с:

```text
server-watch:8080
```

Grafana использует Prometheus как datasource.

---

## Персистентность в Compose

SQLite не хранится внутри контейнера приложения.

В Compose используется отдельный volume:

```text
sqlite_data
```

Он монтируется в:

```text
/data
```

Приложение использует:

```text
DB_PATH=/data/server-watch.db
```

Поэтому удаление и пересоздание контейнера `server-watch` не удаляет историю метрик.

Конфигурация также вынесена из image:

```text
./config.yaml:/app/config.yaml
```

Это необходимо, потому что endpoint `/config` может изменять конфигурацию приложения.

---

## Конфигурация

Путь к конфигурации задаётся через:

```text
CONFIG_PATH
```

Если переменная не задана, используется:

```text
config.yaml
```

Для Docker Compose используется:

```text
CONFIG_PATH=/app/config.yaml
```

Путь к SQLite задаётся через:

```text
DB_PATH
```

---

## Environment variables

В текущей реализации используются следующие переменные окружения:

```text
CONFIG_PATH
DB_PATH
REDIS_ADDR

CPU_THRESHOLD
MEM_THRESHOLD
TRIGGER_COUNT
RESOLVE_COUNT

SLACK_ENABLED
SLACK_URL

PPROF_ENABLED
```

Важно: некоторые переменные из первоначального инфраструктурного плана пока не реализованы в коде, например:

```text
REDIS_PASSWORD
REDIS_DB
SLACK_WEBHOOK_URL
WEBHOOK_URL
LOG_LEVEL
```

Поэтому они не используются текущим deployment'ом.

---

# Kubernetes

Для Kubernetes создан Helm chart:

```text
helm/server-watch/
```

Chart содержит:

```text
Chart.yaml
values.yaml
templates/
```

Проверка chart:

```bash
helm lint helm/server-watch
```

Рендеринг:

```bash
helm template server-watch helm/server-watch
```

---

## Kubernetes resources

Helm chart создаёт:

* Deployment;
* Service;
* ConfigMap;
* Secret;
* PVC для конфигурации;
* PVC для SQLite.

Структура окружения:

```text
              ┌──────────────────┐
              │      Service     │
              │    ClusterIP     │
              │      :8080       │
              └────────┬─────────┘
                       │
                       ▼
              ┌──────────────────┐
              │    Deployment    │
              │  server-watch    │
              └────────┬─────────┘
                       │
              ┌────────┴─────────┐
              ▼                  ▼
       config PVC          sqlite PVC
```

---

## Kubernetes Service

Service имеет тип:

```text
ClusterIP
```

и публикует:

```text
8080
```

наружу сервис напрямую не выставляется.

Для проверки HTTP API внутри кластера использовался временный Pod с `curl`.

Например:

```bash
kubectl run curl --rm -it \
  --image=curlimages/curl \
  --restart=Never \
  -- \
  curl http://server-watch:8080/health
```

---

## Health checks

Для Kubernetes используется endpoint:

```text
/health
```

Он используется для проверки состояния приложения.

Проверяется HTTP response:

```text
200 OK
```

После запуска или пересоздания Pod Kubernetes ждёт, пока приложение станет готово принимать запросы.

Это позволяет не направлять трафик на Pod, который ещё не завершил запуск.

---

# Kubernetes configuration

Начальная конфигурация приложения хранится в:

```text
ConfigMap
```

ConfigMap содержит:

```yaml
cpu_threshold: 80
mem_threshold: 90
trigger_count: 3
resolve_count: 3
slack_enabled: false
slack_url: ""
pprof_enabled: false
```

Так как приложение может изменять `config.yaml` через HTTP API, начальная конфигурация из ConfigMap при запуске копируется в отдельный PVC.

Для этого используется `initContainer`.

Он выполняет логику:

```text
если config.yaml ещё отсутствует
        │
        ▼
скопировать его из ConfigMap
        │
        ▼
основной контейнер использует PVC
```

Это позволяет одновременно использовать:

* ConfigMap для начальной конфигурации;
* PVC для изменяемой конфигурации.

---

# Kubernetes secrets

Для Slack URL используется Kubernetes Secret:

```text
SLACK_URL
```

В Helm chart реальное значение секрета не хранится.

Chart содержит только пустое значение-заглушку.

Реальный secret должен задаваться отдельно при deployment.

Таким образом, секрет не требуется хранить непосредственно в Git-репозитории или Docker image.

---

# SQLite в Kubernetes

SQLite хранится на отдельном PersistentVolumeClaim:

```text
server-watch-sqlite
```

Приложение использует:

```text
DB_PATH=/data/server-watch.db
```

PVC имеет:

```text
ReadWriteOnce
```

и используется одним экземпляром приложения.

Это связано с архитектурным ограничением проекта:

```text
replicaCount: 1
```

Несколько Pod с одной SQLite-базой в текущей архитектуре не используются.

---

# Проверка persistence

Persistence была проверена экспериментально.

Сценарий:

```text
1. Запустить server-watch
2. Записать метрики
3. Получить /history
4. Удалить Pod
5. Дождаться создания нового Pod
6. Снова вызвать /history
```

После пересоздания Pod история метрик сохранилась.

Это подтверждает, что SQLite database действительно находится на PVC, а не внутри filesystem контейнера.

---

# Helm lifecycle

Проверен полный базовый lifecycle Helm release.

Установка:

```bash
helm install server-watch helm/server-watch
```

Обновление:

```bash
helm upgrade server-watch helm/server-watch
```

Откат:

```bash
helm rollback server-watch 1
```

После rollback:

* Pod успешно запустился;
* Service остался доступен;
* PVC остались `Bound`;
* история SQLite сохранилась.

Таким образом, deployment можно изменять и откатывать без потери persistent data.

---

# Graceful shutdown и logging

Предыдущая реализация graceful shutdown сохраняется и при запуске в Kubernetes.

При остановке контейнера Kubernetes отправляет:

```text
SIGTERM
```

Приложение корректно завершает:

* HTTP server;
* background workers;
* collector;
* context-dependent operations.

Логи приложения выводятся в stdout в JSON-формате через:

```text
log/slog
```

Это позволяет использовать стандартные инструменты контейнерной инфраструктуры:

```bash
docker logs
kubectl logs
```

---

# Что проверено

В рамках этапа проверены:

* сборка Docker image;
* запуск контейнера;
* запуск Docker Compose;
* Redis;
* Prometheus;
* Grafana;
* HTTP API внутри Compose;
* Helm lint;
* Helm template;
* Helm install;
* Kubernetes Deployment;
* Kubernetes Service;
* ConfigMap;
* Secret;
* PVC для SQLite;
* PVC для конфигурации;
* `/health` внутри Kubernetes;
* `/history` внутри Kubernetes;
* пересоздание Pod;
* сохранение SQLite history;
* `helm upgrade`;
* `helm rollback`.

---

# Ограничения текущей реализации

Этап специально не доводит инфраструктуру до полноценного production deployment.

Текущие ограничения:

* используется один replica;
* SQLite не рассчитан на несколько экземпляров приложения;
* Docker image примерно 32 MB, а не `<30 MB`;
* runtime использует `scratch`, поэтому внутри контейнера нет shell и диагностических утилит;
* контейнер пока запускается от root;
* Prometheus и Grafana запускаются через Docker Compose, но не входят в Helm chart;
* ServiceMonitor пока не реализован;
* нет CI/CD pipeline;
* нет Ingress и TLS;
* нет service mesh;
* нет Kubernetes-specific exporters вроде `kube-state-metrics`.

Эти ограничения являются осознанными и оставлены для возможных следующих этапов.

---

# Результат этапа

После завершения этапа `server-watch` можно:

* собрать в минимальный Docker image;
* запустить вместе с Redis, Prometheus и Grafana через Docker Compose;
* хранить SQLite и изменяемую конфигурацию вне контейнера;
* передавать конфигурацию через environment variables;
* безопасно передавать Slack URL через Kubernetes Secret;
* установить приложение в Kubernetes через Helm;
* использовать Kubernetes Service для доступа к API;
* сохранять историю метрик при пересоздании Pod;
* выполнять `helm upgrade` и `helm rollback`;
* проверять состояние приложения через `/health`;
* просматривать JSON-логи через стандартные container/Kubernetes tools.

Таким образом, после этапа 7 проект переходит от локального Go-сервиса к воспроизводимому контейнерному и Kubernetes-окружению с базовой персистентностью и управлением deployment lifecycle.
