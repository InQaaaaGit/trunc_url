# Использование документации проекта

> **Примечание**: Данная инструкция актуальна для Go 1.24.0+. В новых версиях Go основной способ работы с документацией - команда `go doc` в терминале.

## Просмотр документации через go doc

**Рекомендуемый способ** - команда `go doc` предоставляет быстрый доступ к документации прямо в терминале.

### Документация пакетов
```bash
# Общая информация о пакете handler
go doc ./internal/handler

# Общая информация о пакете service  
go doc ./internal/service

# Общая информация о пакете storage
go doc ./internal/storage

# Общая информация о пакете config
go doc ./internal/config

# Общая информация о пакете app
go doc ./internal/app
```

### Документация конкретных типов и методов
```bash
# Документация структуры Handler
go doc ./internal/handler.Handler

# Документация интерфейса URLService
go doc ./internal/service.URLService

# Документация интерфейса URLStorage
go doc ./internal/storage.URLStorage

# Документация структуры Config
go doc ./internal/config.Config

# Документация функции NewConfig
go doc ./internal/config.NewConfig

# Документация структуры App
go doc ./internal/app.App
```

### Документация конкретных методов
```bash
# Методы Handler
go doc ./internal/handler.Handler.HandleCreateURL
go doc ./internal/handler.Handler.HandleShortenURL
go doc ./internal/handler.Handler.HandleRedirect

# Методы URLService
go doc ./internal/service.URLServiceImpl.CreateShortURL
go doc ./internal/service.URLServiceImpl.GetOriginalURL
```

## Запуск примеров (Examples)

### Примеры для handler
```bash
# Запустить все примеры handler
go test -v ./internal/handler -run "Example"

# Запустить конкретный пример
go test -v ./internal/handler -run "ExampleHandler_HandleCreateURL"
go test -v ./internal/handler -run "ExampleHandler_HandleShortenURL"
go test -v ./internal/handler -run "ExampleHandler_HandleShortenBatch"
go test -v ./internal/handler -run "ExampleHandler_HandleRedirect"
```

### Примеры для service
```bash
# Запустить все примеры service
go test -v ./internal/service -run "Example"

# Запустить конкретный пример
go test -v ./internal/service -run "ExampleURLService_CreateShortURL"
go test -v ./internal/service -run "ExampleURLService_GetOriginalURL"
go test -v ./internal/service -run "ExampleURLService_CreateShortURLsBatch"
go test -v ./internal/service -run "ExampleNewURLService"
```

## Веб-интерфейс документации

### Вариант 1: Использование pkg.go.dev (только для публичных репозиториев)
```bash
# ⚠️ ВНИМАНИЕ: pkg.go.dev недоступен для приватных репозиториев
# Если проект будет опубликован публично, документация станет доступна на:
# https://pkg.go.dev/github.com/InQaaaaGit/trunc_url.git

# На данный момент используйте локальные методы просмотра документации
```

### Вариант 2: Локальный сервер godoc (для старых версий Go)
```bash
# Установить godoc (если еще не установлен)
go install golang.org/x/tools/cmd/godoc@latest

# Запустить локальный сервер документации
godoc -http=:6060

# Затем открыть в браузере:
# http://localhost:6060/pkg/
```

### Вариант 3: Использование VS Code Go extension
- Установите расширение Go для VS Code
- Используйте команду "Go: Browse Packages" для навигации по документации

## Тестирование

### Запуск всех тестов
```bash
# Запуск всех тестов с покрытием
go test -cover ./...

# Запуск только unit тестов
go test ./internal/...

# Запуск с подробным выводом
go test -v ./...
```

### Проверка покрытия
```bash
# Создание профиля покрытия
go test -coverprofile=coverage.out ./internal/...

# Просмотр покрытия в браузере
go tool cover -html=coverage.out
```

## Примеры использования API

Все примеры доступны в файлах `example_test.go`:
- Создание коротких URL через разные эндпоинты
- Пакетная обработка URL
- Получение и удаление URL пользователя
- Работа с сервисным слоем

Примеры демонстрируют правильное использование API и могут служить основой для интеграции. 