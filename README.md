[![pipeline status](https://gitlab.com/devpro_studio/FeatureChaos/badges/main/pipeline.svg)](https://gitlab.com/devpro_studio/FeatureChaos/-/commits/main)

[![Latest Release](https://gitlab.com/devpro_studio/FeatureChaos/-/badges/release.svg)](https://gitlab.com/devpro_studio/FeatureChaos/-/releases)

## FeatureChaos — управление флагами фич и процентными выкатками

FeatureChaos — это легковесный сервис для управления фичами, процентными выкатками и таргетингом по атрибутам. Используется для внедрения TBD (Trunk Based Development). Он предоставляет:

- gRPC API со стримингом изменений конфигурации в реальном времени
- HTTP Admin API и простую UI-страницу для управления
- Хранилище в Postgres, кэширование, сбор статистики использования
- SDK для Go, Python и PHP для простой интеграции в сервисы

### Ключевые возможности

- Процентное включение на уровне фичи: 0–100%.
- Таргетинг по ключам и значениям:
  - Процент на уровне ключа (например, country=\*)
  - Точный процент на уровне конкретного значения ключа (например, country=US → 50%)
- Стрим обновлений: клиенты получают только новые версии конфигурации без опросов.
- Реактивная система работы с фичами: вычисление процентов и включение/выключение фич происходит в реальном времени на стороне клиента.
- Сбор статистики использования фич (auto-send из SDK) и защита от удаления активных сущностей.
- Привязка фич к сервисам: разграничение доступа «какие сервисы видят какую фичу».
- Простое администрирование через HTTP API и встроенную страницу `/`.

### Преимущества

- Простота интеграции (готовые SDK, минимальные зависимости)
- Низкая нагрузка на сервер за счёт стриминга обновлений
- Стабильная раскатка (детерминированное распределение по бакетам)
- Масштабируемая архитектура (Postgres + кэш, отдельные репозитории и сервисы)

## Быстрый старт

### Требования

- Postgres 13+
- Go 1.23+ (рекомендуется 1.24)
- Redis (для кэширования)

### Конфигурация

Скопируйте пример в рабочий файл:

```bash
cp cfg_example.yaml cfg.yaml
```

В `cfg.yaml` задайте подключения:

```yaml
engine:
  - type: logger
    name: std
    level: DEBUG
    enable: true
  - type: cache
    name: primary # Redis (опционально)
    hosts: "127.0.0.1:6379"
  - type: cache
    name: secondary # In-memory
    time_clear: 60s
    shard_count: 5
  - type: database
    name: primary # Postgres DSN
    uri: "postgres://postgres:passwd@127.0.0.1:5432/features"
  - type: server
    name: grpc
    port: 9090
  - type: server
    name: http
    port: 8080
```

### Миграции БД (goose)

В репозитории есть `Makefile`, который ожидает `.env` с параметрами подключения:

```env
DB_USER=postgres
DB_PASSWORD=passwd
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=features
DB_SSLMODE=disable
```

Применить миграции:

```bash
make goose-up
# проверить статус
make goose-status
```

### Запуск локально (Go)

```bash
go build -o FeatureChaos ./cmd/app/main.go
./FeatureChaos   # читает cfg.yaml из текущей директории
```

HTTP админка: `http://localhost:8080/`

gRPC: `localhost:9090`

### Запуск в Docker

```bash
docker build -t featurechaos .
# cfg.yaml должен быть рядом и смонтирован внутрь контейнера
# Postgres должен быть доступен из контейнера (проверьте uri в cfg.yaml)
docker run --rm -p 8080:8080 -p 9090:9090 \
  -v "$PWD/cfg.yaml:/app/cfg.yaml:ro" \
  featurechaos
```

При необходимости миграции можно выполнить внутри контейнера (в образ включён `goose`):

```bash
docker run --rm -v "$PWD/cfg.yaml:/app/cfg.yaml:ro" featurechaos \
  goose -dir /app/migrations \
  postgres "user=postgres password=passwd host=host.docker.internal port=5432 dbname=features sslmode=disable" up
```

## HTTP Admin API (кратко)

OpenAPI: `openapi.yaml`. Ниже — основные операции с примерами.

- Создать фичу

```bash
curl -X POST http://localhost:8080/api/features \
  -H 'Content-Type: application/json' \
  -d '{"name":"search", "description":"New search UI"}'
```

- Обновить фичу

```bash
curl -X PUT http://localhost:8080/api/features/{featureId} \
  -H 'Content-Type: application/json' \
  -d '{"name":"search", "description":"v2"}'
```

- Удалить фичу

```bash
curl -X DELETE http://localhost:8080/api/features/{featureId}
```

- Установить процент на уровне фичи (0–100)

```bash
curl -X POST http://localhost:8080/api/features/{featureId}/value \
  -H 'Content-Type: application/json' \
  -d '{"value": 25}'
```

- Создать ключ (например, "country") для фичи

```bash
curl -X POST http://localhost:8080/api/features/{featureId}/keys \
  -H 'Content-Type: application/json' \
  -d '{"key": "country", "description": "geo targeting"}'
```

- Установить процент на уровне ключа

```bash
curl -X POST http://localhost:8080/api/keys/{keyId}/value \
  -H 'Content-Type: application/json' \
  -d '{"value": 50}'
```

- Создать параметр (значение ключа, например "US")

```bash
curl -X POST http://localhost:8080/api/keys/{keyId}/params \
  -H 'Content-Type: application/json' \
  -d '{"name": "US"}'
```

- Установить процент для конкретного значения ключа

```bash
curl -X POST http://localhost:8080/api/params/{paramId}/value \
  -H 'Content-Type: application/json' \
  -d '{"value": 75}'
```

- Управление сервисами и доступами

```bash
# создать сервис
curl -X POST http://localhost:8080/api/services -H 'Content-Type: application/json' -d '{"name":"billing"}'
# привязать фичу к сервису
curl -X POST http://localhost:8080/api/features/{featureId}/services/{serviceId}
# снять привязку
curl -X DELETE http://localhost:8080/api/features/{featureId}/services/{serviceId}
```

Страница администрирования доступна по `GET /` и использует те же API.

## gRPC API

Прото-файл: `proto/FeatureChaos.proto`.

- Метод `Subscribe(GetAllFeatureRequest) returns (stream GetFeatureResponse)` — серверный стрим обновлений конфигурации для сервиса с учётом версии.
- Метод `Stats(stream SendStatsRequest) returns (google.protobuf.Empty)` — клиентский стрим событий использования фич.

### Go SDK (рекомендуется)

Модуль: `sdk/featurechaos` (`go get gitlab.com/devpro_studio/featurechaos-sdk@latest`). Пример:

```go
package main

import (
  "context"
  "fmt"
  fc "gitlab.com/devpro_studio/featurechaos-sdk"
)

func main() {
  ctx := context.Background()
  client, err := fc.New(ctx, "127.0.0.1:9090", "billing", fc.Options{AutoSendStats: true})
  if err != nil { panic(err) }
  defer client.Close()

  enabled := client.IsEnabled("search", "user-123", map[string]string{"country":"US"})
  fmt.Println("search enabled:", enabled)
}
```

### Python SDK

Реализация в `sdk/python/featurechaos/client.py` (standalone, без сгенерированных pb).

```python
from featurechaos.client import FeatureChaosClient, Options

cli = FeatureChaosClient("127.0.0.1:9090", "billing", Options(auto_send_stats=True))
try:
    is_on = cli.is_enabled("search", "user-123", {"country": "US"})
    print("search enabled:", is_on)
finally:
    cli.close()
```

> Примечание: убедитесь, что путь `sdk/python` доступен в `PYTHONPATH`, либо упакуйте SDK под свои нужды.

## Как принимается решение включения фичи

Порядок приоритета процентов:

1. Точное совпадение пары ключ=значение → процент из параметра
2. Если (1) не найден — процент на уровне ключа
3. Если (2) не задан — процент на уровне фичи
4. Процент клэмпится в диапазон 0..100. При `0` — всегда выключено, при `100` — всегда включено.
   Распределение стабильное относительно пары `(featureName, seed)` с использованием быстрых хешей, чтобы один и тот же пользователь стабильно попадал в свою группу.

## Статистика

- SDK по умолчанию отправляет события использования (можно отключить `AutoSendStats=false` / `auto_send_stats=False`).
- Сервис хранит агрегаты, использует их для индикации активности и блокировки удаления активных фич/сервисов.

## Безопасность и развёртывание

- Admin API не содержит встроенной аутентификации — рекомендовано размещать за обратным прокси с аутентификацией и TLS, ограничить доступ сетью.
- Храните `cfg.yaml` и секреты отдельно от образа, монтируйте их при запуске.
- Следите за резервным копированием Postgres.

## Лицензия

См. `LICENSE.txt`.
