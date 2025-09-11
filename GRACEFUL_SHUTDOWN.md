# Graceful Shutdown Implementation

## Описание

Реализован механизм graceful shutdown для сервера сокращения URL, который обеспечивает корректное завершение работы сервера по системным сигналам и сохранение всех данных.

## Поддерживаемые сигналы

Сервер корректно завершается по следующим сигналам:
- `syscall.SIGTERM` - сигнал завершения (kill)
- `syscall.SIGINT` - сигнал прерывания (Ctrl+C)
- `syscall.SIGQUIT` - сигнал выхода (Ctrl+\)

## Функциональность

### 1. Обработка сигналов
- Настроена обработка системных сигналов в `cmd/shortener/main.go`
- Сервер запускается в отдельной горутине для неблокирующего ожидания сигналов
- При получении сигнала инициируется процедура graceful shutdown

### 2. Завершение сервера
- Используется контекст с таймаутом 30 секунд для завершения работы
- Сервер дожидается завершения всех активных HTTP-запросов
- Реализовано в методе `HTTPServer.Shutdown()`

### 3. Сохранение данных
- Все хранилища (File, PostgreSQL, Memory) реализуют интерфейс `storage.Closer`
- Файловое хранилище принудительно синхронизирует данные с диском перед закрытием
- PostgreSQL корректно закрывает соединения с базой данных
- Memory storage освобождает ресурсы

## Архитектура

### Новые интерфейсы

```go
// storage/interface.go
type Closer interface {
    Close() error
}

// server/server.go
type Starter interface {
    Start() error
    Shutdown(ctx context.Context) error
}
```

### Модифицированные компоненты

1. **HTTPServer** (`internal/server/server.go`)
   - Добавлен метод `Shutdown()` для корректного завершения
   - Интегрирован с `http.Server.Shutdown()`

2. **URLService** (`internal/service/url.go`)
   - Добавлен метод `Close()` в интерфейс и реализацию
   - Проверяет, поддерживает ли хранилище интерфейс `Closer`

3. **App** (`internal/app/app.go`)
   - Добавлен метод `Close()` для закрытия сервиса
   - Сохраняет ссылку на сервис для корректного завершения

4. **Storage implementations**
   - `FileStorage`: добавлена синхронизация данных перед закрытием
   - `PostgresStorage`: корректное закрытие соединений
   - `MemoryStorage`: пустая реализация для совместимости

## Последовательность shutdown

1. **Получение сигнала** - сервер получает SIGTERM/SIGINT/SIGQUIT
2. **Остановка приема новых запросов** - HTTP сервер перестает принимать новые соединения
3. **Ожидание завершения активных запросов** - все текущие запросы обрабатываются до конца
4. **Закрытие HTTP сервера** - освобождение сетевых ресурсов
5. **Сохранение данных** - синхронизация файлового хранилища, закрытие БД
6. **Завершение процесса** - корректный выход из приложения

## Таймауты

- **Server shutdown**: 30 секунд - максимальное время ожидания завершения запросов
- **Context timeout**: graceful shutdown должен завершиться в указанное время
- При превышении таймаута принудительное завершение с логированием ошибки

## Тестирование

Создан тест `TestGracefulShutdown` в `cmd/shortener/graceful_shutdown_test.go`:
- Запуск сервера
- Проверка работоспособности
- Имитация graceful shutdown
- Проверка корректного закрытия всех ресурсов

## Использование

```bash
# Запуск сервера
./shortener

# Graceful shutdown
kill -TERM <PID>
# или
kill -INT <PID>  # Ctrl+C
# или  
kill -QUIT <PID>  # Ctrl+\
```

## Логирование

Процесс graceful shutdown полностью логируется:
- Получение сигнала с указанием типа
- Начало процедуры shutdown
- Закрытие сервера
- Сохранение данных в хранилище
- Успешное/неуспешное завершение каждого этапа

Пример логов:
```
INFO    Received shutdown signal        {"signal": "interrupt"}
INFO    Initiating graceful shutdown...
INFO    Shutting down server...
INFO    Server shutdown completed
INFO    Closing application...
INFO    Closing URL service...
INFO    Storage closed successfully
INFO    Application closed successfully
``` 