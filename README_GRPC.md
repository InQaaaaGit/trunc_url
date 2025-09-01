# gRPC Поддержка для Сервиса Сокращения URL

## Обзор

Сервис сокращения URL теперь поддерживает gRPC протокол в дополнение к HTTP API. Все существующие HTTP обработчики доступны через gRPC и функционируют идентично.

## Архитектура

### Принцип работы

gRPC и HTTP обработчики работают как фасады к общему слою бизнес-логики (`service.URLService`). Это обеспечивает:

- **Единообразие**: Одинаковая логика обработки для HTTP и gRPC
- **Переиспользование кода**: Общий сервисный слой
- **Простота поддержки**: Изменения в бизнес-логике автоматически применяются к обоим протоколам

### Структура

```
┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │    │   gRPC Client   │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          ▼                      ▼
┌─────────────────┐    ┌─────────────────┐
│  HTTP Handler   │    │  gRPC Handler   │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          └──────────┬───────────┘
                     ▼
            ┌─────────────────┐
            │ URLService      │
            │ (Business Logic)│
            └─────────┬───────┘
                      ▼
            ┌─────────────────┐
            │   Storage       │
            │   (Database)    │
            └─────────────────┘
```

## Конфигурация

### Параметры gRPC

Добавлены новые параметры конфигурации:

```json
{
    "grpc_server_address": "localhost:9090",
    "enable_grpc": true
}
```

### Переменные окружения

- `GRPC_SERVER_ADDRESS` - адрес для запуска gRPC сервера (по умолчанию: `:9090`)
- `ENABLE_GRPC` - включить gRPC сервер (по умолчанию: `false`)

### Флаги командной строки

- `-grpc-addr` - адрес запуска gRPC сервера
- `-grpc` - включить gRPC сервер

## API Endpoints

### 1. CreateShortURL

**gRPC метод**: `CreateShortURL`

**Описание**: Создает короткий URL из оригинального

**Запрос**:
```protobuf
message CreateShortURLRequest {
    string original_url = 1;
}
```

**Ответ**:
```protobuf
message CreateShortURLResponse {
    string short_url = 1;
    int32 status_code = 2;
    string error_message = 3;
}
```

### 2. GetOriginalURL

**gRPC метод**: `GetOriginalURL`

**Описание**: Получает оригинальный URL по короткому идентификатору

**Запрос**:
```protobuf
message GetOriginalURLRequest {
    string short_id = 1;
}
```

**Ответ**:
```protobuf
message GetOriginalURLResponse {
    string original_url = 1;
    int32 status_code = 2;
    string error_message = 3;
}
```

### 3. CreateShortURLsBatch

**gRPC метод**: `CreateShortURLsBatch`

**Описание**: Создает несколько коротких URL за один запрос

**Запрос**:
```protobuf
message CreateShortURLsBatchRequest {
    repeated BatchRequestEntry entries = 1;
}

message BatchRequestEntry {
    string correlation_id = 1;
    string original_url = 2;
}
```

**Ответ**:
```protobuf
message CreateShortURLsBatchResponse {
    repeated BatchResponseEntry entries = 1;
    int32 status_code = 2;
    string error_message = 3;
}

message BatchResponseEntry {
    string correlation_id = 1;
    string short_url = 2;
}
```

### 4. GetUserURLs

**gRPC метод**: `GetUserURLs`

**Описание**: Получает все URL пользователя

**Запрос**:
```protobuf
message GetUserURLsRequest {
    string user_id = 1;
}
```

**Ответ**:
```protobuf
message GetUserURLsResponse {
    repeated UserURL urls = 1;
    int32 status_code = 2;
    string error_message = 3;
}

message UserURL {
    string short_url = 1;
    string original_url = 2;
}
```

### 5. BatchDeleteURLs

**gRPC метод**: `BatchDeleteURLs`

**Описание**: Помечает URL как удаленные

**Запрос**:
```protobuf
message BatchDeleteURLsRequest {
    repeated string short_urls = 1;
    string user_id = 2;
}
```

**Ответ**:
```protobuf
message BatchDeleteURLsResponse {
    int32 status_code = 1;
    string error_message = 2;
}
```

### 6. Ping

**gRPC метод**: `Ping`

**Описание**: Проверяет соединение с хранилищем

**Запрос**:
```protobuf
message PingRequest {}
```

**Ответ**:
```protobuf
message PingResponse {
    int32 status_code = 1;
    string error_message = 2;
}
```

### 7. GetStats

**gRPC метод**: `GetStats`

**Описание**: Возвращает статистику сервиса

**Запрос**:
```protobuf
message GetStatsRequest {}
```

**Ответ**:
```protobuf
message GetStatsResponse {
    int32 urls_count = 1;
    int32 users_count = 2;
    int32 status_code = 3;
    string error_message = 4;
}
```

## HTTP Adapter

Для удобства тестирования и отладки предоставляется HTTP адаптер для gRPC методов:

### Endpoints

- `POST /grpc/create?url=<original_url>` - создание короткого URL
- `GET /grpc/get?id=<short_id>` - получение оригинального URL
- `GET /grpc/ping` - проверка соединения
- `GET /grpc/stats` - получение статистики

### Примеры использования

```bash
# Создание короткого URL
curl -X POST "http://localhost:8080/grpc/create?url=https://example.com"

# Получение оригинального URL
curl "http://localhost:8080/grpc/get?id=abc123"

# Проверка соединения
curl "http://localhost:8080/grpc/ping"

# Получение статистики
curl "http://localhost:8080/grpc/stats"
```

## Запуск сервера

### Только HTTP
```bash
./api -config config.json
```

### HTTP + gRPC
```bash
./api -config config.json -grpc -grpc-addr :9090
```

### Только gRPC (через конфигурацию)
```json
{
    "server_address": "",
    "enable_grpc": true,
    "grpc_server_address": ":9090"
}
```

## Логирование

gRPC сервер использует тот же логгер, что и HTTP сервер. Все запросы логируются с информацией о методе и параметрах.

### Пример логов

```
INFO    gRPC request started    {"method": "/urlservice.URLService/CreateShortURL", "request": {"original_url": "https://example.com"}}
INFO    Received URL in gRPC CreateShortURL request    {"url": "https://example.com"}
INFO    gRPC request completed    {"method": "/urlservice.URLService/CreateShortURL"}
```

## Обработка ошибок

gRPC методы возвращают HTTP статус коды в поле `status_code`:

- `200` - успешное выполнение
- `400` - неверный запрос
- `404` - URL не найден
- `409` - конфликт (URL уже существует)
- `410` - URL удален
- `500` - внутренняя ошибка сервера
- `503` - сервис недоступен

## Тестирование

### Unit тесты

```bash
go test ./internal/grpc/...
```

### Интеграционные тесты

```bash
# Запуск сервера с gRPC
./api -grpc -grpc-addr :9090

# Тестирование через HTTP адаптер
curl -X POST "http://localhost:8080/grpc/create?url=https://example.com"
curl "http://localhost:8080/grpc/get?id=<short_id>"
```

## Производительность

gRPC обеспечивает лучшую производительность по сравнению с HTTP JSON API за счет:

- Бинарного протокола (Protocol Buffers)
- HTTP/2 мультиплексирования
- Сжатия заголовков
- Потоковой передачи данных

## Безопасность

gRPC сервер поддерживает те же механизмы безопасности, что и HTTP сервер:

- Проверка доверенной подсети для внутренних API
- Аутентификация пользователей
- Валидация входных данных

## Совместимость

- Все существующие HTTP API продолжают работать
- gRPC API полностью совместим с HTTP API
- Можно использовать оба протокола одновременно
- Плавная миграция с HTTP на gRPC

## Ограничения

- gRPC требует HTTP/2
- Некоторые прокси могут не поддерживать gRPC
- Отладка gRPC запросов сложнее, чем HTTP
- Для веб-браузеров требуется gRPC-Web прокси 