# HTTPS поддержка в сервисе сокращения URL

## Обзор

В проект добавлена поддержка HTTPS с использованием TLS сертификатов. Сервер может работать как в HTTP, так и в HTTPS режиме в зависимости от конфигурации.

## Конфигурация

### Переменные окружения

```bash
export ENABLE_HTTPS=true           # Включить HTTPS сервер
export TLS_CERT_FILE=server.crt    # Путь к файлу сертификата (по умолчанию: server.crt)
export TLS_KEY_FILE=server.key     # Путь к файлу приватного ключа (по умолчанию: server.key)
```

### Флаги командной строки

```bash
go run ./cmd/shortener -s true                    # Включить HTTPS
go run ./cmd/shortener -tls-cert server.crt       # Указать путь к сертификату
go run ./cmd/shortener -tls-key server.key        # Указать путь к ключу
```

## Генерация сертификатов для разработки

### Linux/macOS

```bash
# Сделать скрипт исполняемым
chmod +x generate_certs.sh

# Запустить генерацию сертификатов
./generate_certs.sh
```

### Windows (PowerShell)

```powershell
# Запустить генерацию сертификатов
.\generate_certs.ps1
```

### Ручная генерация через OpenSSL

```bash
# Генерация приватного ключа
openssl genrsa -out server.key 2048

# Генерация самоподписанного сертификата
openssl req -new -x509 -key server.key -out server.crt -days 365 \
  -subj "/C=RU/ST=Moscow/L=Moscow/O=TruncURL/OU=Dev/CN=localhost"
```

## Запуск сервера

### HTTP режим (по умолчанию)

```bash
go run ./cmd/shortener
# или
go run ./cmd/api
```

### HTTPS режим

```bash
# Через флаг
go run ./cmd/shortener -s true

# Через переменную окружения
export ENABLE_HTTPS=true
go run ./cmd/shortener

# С указанием пути к сертификатам
go run ./cmd/shortener -s true -tls-cert /path/to/server.crt -tls-key /path/to/server.key
```

## Примеры использования

### Тестирование HTTPS сервера

```bash
# Запуск сервера
export ENABLE_HTTPS=true
go run ./cmd/shortener

# Тестирование с curl (игнорируя самоподписанный сертификат)
curl -k -X POST https://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# Тестирование через браузер
# Откройте https://localhost:8080 (появится предупреждение о сертификате)
```

### Production конфигурация

```bash
# Использование реальных сертификатов (например, Let's Encrypt)
export ENABLE_HTTPS=true
export TLS_CERT_FILE=/etc/ssl/certs/yourdomain.crt
export TLS_KEY_FILE=/etc/ssl/private/yourdomain.key
export SERVER_ADDRESS=:443
go run ./cmd/shortener
```

## Безопасность

### Для разработки
- Используйте самоподписанные сертификаты только для разработки
- Браузеры будут показывать предупреждения о безопасности
- Используйте флаг `-k` в curl для игнорирования ошибок сертификата

### Для production
- Используйте сертификаты от доверенного CA (например, Let's Encrypt)
- Настройте правильные DNS записи для вашего домена
- Рассмотрите использование HTTP Strict Transport Security (HSTS)
- Регулярно обновляйте сертификаты

## Troubleshooting

### Ошибка "certificate signed by unknown authority"

```bash
# Для тестирования используйте флаг -k в curl
curl -k https://localhost:8080

# Или добавьте сертификат в доверенные (не рекомендуется для production)
```

### Ошибка "bind: address already in use"

```bash
# Проверьте, что порт не занят
netstat -tlnp | grep :8080

# Или используйте другой порт
go run ./cmd/shortener -s true -a :8443
```

### Ошибка "no such file or directory" для сертификатов

```bash
# Убедитесь, что файлы сертификатов существуют
ls -la server.crt server.key

# Или сгенерируйте их заново
./generate_certs.sh
```

## Архитектурные изменения

### Добавленные поля в Config

```go
type Config struct {
    // ... существующие поля ...
    
    // HTTPS настройки
    EnableHTTPS string `env:"ENABLE_HTTPS"`  // Включить HTTPS сервер
    TLSCertFile string `env:"TLS_CERT_FILE"` // Путь к файлу сертификата TLS
    TLSKeyFile  string `env:"TLS_KEY_FILE"`  // Путь к файлу приватного ключа TLS
}
```

### Новые методы

```go
// IsHTTPSEnabled проверяет, включен ли HTTPS режим
func (c *Config) IsHTTPSEnabled() bool
```

### Измененные флаги

- `-s` теперь используется для включения HTTPS (вместо секретного ключа)
- `-secret-key` новый флаг для секретного ключа
- `-tls-cert` путь к TLS сертификату
- `-tls-key` путь к TLS ключу 