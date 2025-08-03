# Реализация HTTPS поддержки

## ✅ Выполненные требования

В соответствии с заданием для трека «Сервис сокращения URL» добавлена возможность включения HTTPS в веб-сервере:

- ✅ **Флаг -s**: При передаче флага `-s` запускается HTTPS сервер
- ✅ **Переменная окружения ENABLE_HTTPS**: При установке переменной `ENABLE_HTTPS` запускается HTTPS сервер  
- ✅ **ListenAndServeTLS**: Используется метод `http.ListenAndServeTLS` для HTTPS сервера

## 🔧 Архитектурные изменения

### 1. Config (internal/config/config.go)

**Добавленные поля:**
```go
// HTTPS настройки
EnableHTTPS string `env:"ENABLE_HTTPS"` // Включить HTTPS сервер
TLSCertFile string `env:"TLS_CERT_FILE"` // Путь к файлу сертификата TLS
TLSKeyFile  string `env:"TLS_KEY_FILE"`  // Путь к файлу приватного ключа TLS
```

**Новые методы:**
```go
// IsHTTPSEnabled проверяет, включен ли HTTPS режим
func (c *Config) IsHTTPSEnabled() bool
```

**Обновленные флаги:**
- `-s string` - включить HTTPS сервер (изменен с секретного ключа)
- `-secret-key string` - секретный ключ для подписи кук (новый флаг)
- `-tls-cert string` - путь к файлу TLS сертификата  
- `-tls-key string` - путь к файлу TLS приватного ключа

### 2. App (internal/app/app.go)

**Обновленный метод Run():**
```go
func (a *App) Run() error {
    // ...
    if a.config.IsHTTPSEnabled() {
        a.logger.Info("Starting HTTPS server", ...)
        return server.ListenAndServeTLS(a.config.TLSCertFile, a.config.TLSKeyFile)
    }
    
    a.logger.Info("Starting HTTP server", ...)
    return server.ListenAndServe()
}
```

### 3. Main files (cmd/shortener/main.go, cmd/api/main.go)

**Добавлена логика выбора протокола:**
```go
if cfg.IsHTTPSEnabled() {
    // HTTPS запуск
    server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
} else {
    // HTTP запуск
    server.ListenAndServe()
}
```

## 🛠 Утилиты для разработки

### 1. Генерация сертификатов

**Linux/macOS:**
```bash
./generate_certs.sh
```

**Windows:**
```powershell
.\generate_certs.ps1
```

### 2. Значения по умолчанию

- `TLSCertFile`: `"server.crt"`
- `TLSKeyFile`: `"server.key"`
- `EnableHTTPS`: `""` (отключен)

## 📋 Способы запуска

### 1. Через флаг командной строки

```bash
# Включить HTTPS
go run ./cmd/shortener -s true

# С пользовательскими сертификатами
go run ./cmd/shortener -s true -tls-cert my.crt -tls-key my.key
```

### 2. Через переменные окружения

```bash
# Включить HTTPS
export ENABLE_HTTPS=true
go run ./cmd/shortener

# С пользовательскими сертификатами
export ENABLE_HTTPS=true
export TLS_CERT_FILE=my.crt
export TLS_KEY_FILE=my.key
go run ./cmd/shortener
```

### 3. Смешанный режим

```bash
# Переменная окружения + флаги
export ENABLE_HTTPS=true
go run ./cmd/shortener -tls-cert custom.crt -tls-key custom.key
```

## 🧪 Тестирование

### 1. Unit тесты

```bash
go test ./internal/config -v -run="TestHTTPS"
```

### 2. Интеграционное тестирование

```bash
# Запуск HTTPS сервера
export ENABLE_HTTPS=true
go run ./cmd/shortener

# Тестирование с curl
curl -k -X POST https://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'
```

## 🔒 Безопасность

### Для разработки
- Используются самоподписанные сертификаты
- Браузеры показывают предупреждения о безопасности
- Подходит только для локальной разработки

### Для production
- Используйте сертификаты от доверенного CA
- Настройте правильные DNS записи
- Регулярно обновляйте сертификаты

## 📁 Файловая структура

```
trunc_url/
├── cmd/
│   ├── shortener/main.go    # HTTPS поддержка
│   └── api/main.go          # HTTPS поддержка
├── internal/
│   ├── config/
│   │   ├── config.go        # HTTPS конфигурация
│   │   └── config_https_test.go # HTTPS тесты
│   └── app/app.go           # HTTPS логика запуска
├── generate_certs.sh        # Генерация сертификатов (Linux/macOS)
├── generate_certs.ps1       # Генерация сертификатов (Windows)
├── README_HTTPS.md          # Подробная документация
└── .gitignore               # Исключение сертификатов
```

## ✅ Проверка соответствия заданию

1. ✅ **Флаг -s**: Реализован для включения HTTPS
2. ✅ **Переменная ENABLE_HTTPS**: Реализована для включения HTTPS
3. ✅ **http.ListenAndServeTLS**: Используется для HTTPS сервера
4. ✅ **Совместимость**: HTTP режим остался без изменений
5. ✅ **Конфигурируемость**: Пути к сертификатам настраиваются через флаги/переменные
6. ✅ **Документация**: Полная документация с примерами
7. ✅ **Тесты**: Unit тесты для HTTPS конфигурации

Реализация полностью соответствует требованиям задания! 🎉 