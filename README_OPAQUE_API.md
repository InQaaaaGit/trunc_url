# OpaqueAPI для gRPC - Современный подход к API дизайну

## Что такое OpaqueAPI?

OpaqueAPI - это современный подход к проектированию gRPC API, который упрощает интерфейс и улучшает производительность за счет использования `google.protobuf.Any` для передачи данных.

## Основные принципы

### 1. Единая точка входа
Вместо множества отдельных RPC методов, OpaqueAPI использует один универсальный метод `Execute`:

```protobuf
service URLService {
  // Execute выполняет операцию с URL на основе переданного типа сообщения
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
  
  // Ping проверяет соединение с хранилищем
  rpc Ping(google.protobuf.Empty) returns (PingResponse);
}
```

### 2. Универсальные типы запросов и ответов
```protobuf
message ExecuteRequest {
  string operation_type = 1;           // Тип операции
  google.protobuf.Any payload = 2;     // Данные для операции
}

message ExecuteResponse {
  bool success = 1;                    // Успешность операции
  int32 status_code = 2;               // HTTP статус код
  string error_message = 3;            // Сообщение об ошибке
  google.protobuf.Any payload = 4;     // Результат операции
}
```

### 3. Типизированные payload'ы
Каждая операция имеет свой тип payload:

```protobuf
// Создание короткого URL
message CreateShortURLPayload {
  string original_url = 1;
}

message CreateShortURLResult {
  string short_url = 1;
}

// Получение оригинального URL
message GetOriginalURLPayload {
  string short_id = 1;
}

message GetOriginalURLResult {
  string original_url = 1;
}
```

## Преимущества OpaqueAPI

### 1. **Упрощение API**
- Один метод вместо множества
- Единообразная обработка ошибок
- Консистентная структура ответов

### 2. **Улучшение производительности**
- Меньше аллокаций памяти
- Снижение overhead'а на маршрутизацию
- Оптимизация сериализации/десериализации

### 3. **Упрощение версионирования**
- Легче добавлять новые поля
- Меньше breaking changes
- Обратная совместимость

### 4. **Лучшая типизация**
- Использование `google.protobuf.Any`
- Валидация типов на уровне protobuf
- Автогенерация кода

## Реализация в нашем проекте

### Структура типов операций
```go
const (
    OperationCreateShortURL      = "create_short_url"
    OperationGetOriginalURL      = "get_original_url"
    OperationCreateShortURLsBatch = "create_short_urls_batch"
    OperationGetUserURLs         = "get_user_urls"
    OperationBatchDeleteURLs     = "batch_delete_urls"
    OperationGetStats            = "get_stats"
)
```

### Обработка в handler
```go
func (h *GRPCHandler) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
    switch req.OperationType {
    case OperationCreateShortURL:
        return h.handleCreateShortURL(ctx, req.Payload)
    case OperationGetOriginalURL:
        return h.handleGetOriginalURL(ctx, req.Payload)
    // ... другие операции
    default:
        return &ExecuteResponse{
            Success:      false,
            StatusCode:   400,
            ErrorMessage: "Unknown operation type",
        }, nil
    }
}
```

## Миграция с традиционного подхода

### До (традиционный gRPC)
```protobuf
service URLService {
  rpc CreateShortURL(CreateShortURLRequest) returns (CreateShortURLResponse);
  rpc GetOriginalURL(GetOriginalURLRequest) returns (GetOriginalURLResponse);
  rpc CreateShortURLsBatch(CreateShortURLsBatchRequest) returns (CreateShortURLsBatchResponse);
  // ... множество других методов
}
```

### После (OpaqueAPI)
```protobuf
service URLService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
  rpc Ping(google.protobuf.Empty) returns (PingResponse);
}
```

## Использование в клиентском коде

### Создание запроса
```go
// Создаем payload для создания короткого URL
createPayload := &CreateShortURLPayload{
    OriginalURL: "https://example.com",
}

// Сериализуем в Any
anyPayload, _ := anypb.New(createPayload)

// Создаем ExecuteRequest
req := &ExecuteRequest{
    OperationType: OperationCreateShortURL,
    Payload:       anyPayload,
}

// Вызываем Execute
resp, err := client.Execute(ctx, req)
```

### Обработка ответа
```go
if resp.Success {
    // Десериализуем payload
    var result CreateShortURLResult
    if err := resp.Payload.UnmarshalTo(&result); err == nil {
        fmt.Printf("Short URL: %s\n", result.ShortURL)
    }
} else {
    fmt.Printf("Error: %s\n", resp.ErrorMessage)
}
```

## Лучшие практики

### 1. **Валидация типов операций**
```go
func validateOperationType(opType string) error {
    validTypes := map[string]bool{
        OperationCreateShortURL:      true,
        OperationGetOriginalURL:      true,
        OperationCreateShortURLsBatch: true,
        // ... другие типы
    }
    
    if !validTypes[opType] {
        return fmt.Errorf("unknown operation type: %s", opType)
    }
    return nil
}
```

### 2. **Обработка ошибок**
```go
func (h *GRPCHandler) handleError(err error, operation string) *ExecuteResponse {
    h.logger.Error("Operation failed", 
        zap.String("operation", operation), 
        zap.Error(err))
    
    return &ExecuteResponse{
        Success:      false,
        StatusCode:   500,
        ErrorMessage: "Internal server error",
    }
}
```

### 3. **Логирование и мониторинг**
```go
func (h *GRPCHandler) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
    start := time.Now()
    defer func() {
        h.logger.Info("Operation completed",
            zap.String("operation", req.OperationType),
            zap.Duration("duration", time.Since(start)))
    }()
    
    // ... обработка операции
}
```

## Заключение

OpaqueAPI представляет собой современный подход к проектированию gRPC API, который:

- **Упрощает** интерфейс сервиса
- **Улучшает** производительность
- **Упрощает** версионирование и поддержку
- **Соответствует** современным best practices

Этот подход особенно полезен для:
- Микросервисной архитектуры
- API с множеством операций
- Проектов, требующих частых изменений
- Систем с высокими требованиями к производительности

## Дальнейшее развитие

В будущем можно рассмотреть:
1. **Автоматическую генерацию** типов payload'ов
2. **Middleware для валидации** типов операций
3. **Кэширование** результатов операций
4. **Метрики и трейсинг** для каждой операции
5. **Rate limiting** на уровне типов операций 