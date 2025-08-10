# JSON конфигурация приложения

## Обзор

В рамках инкремента 22 добавлена возможность конфигурации приложения с помощью JSON файла. Все действующие опции приложения поддерживаются в JSON формате.

## Приоритет конфигурации

Конфигурация применяется в следующем порядке приоритета (от высшего к низшему):

1. **Флаги командной строки** (наивысший приоритет)
2. **Переменные окружения** (средний приоритет)  
3. **JSON файл конфигурации** (низший приоритет)

## Указание файла конфигурации

### Через флаги командной строки

```bash
# Короткий флаг
go run ./cmd/shortener -c config.json

# Длинный флаг  
go run ./cmd/shortener -config config.json
```

### Через переменную окружения

```bash
export CONFIG=config.json
go run ./cmd/shortener
```

## Формат JSON файла

### Полная структура

```json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost",
    "file_storage_path": "/path/to/file.db",
    "database_dsn": "",
    "enable_https": true,
    "tls_cert_file": "server.crt",
    "tls_key_file": "server.key",
    "secret_key": "your-custom-secret-key",
    "batch_delete_max_workers": 5,
    "batch_delete_batch_size": 10,
    "batch_delete_sequential_threshold": 8
}
```

### Минимальная конфигурация

```json
{
    "server_address": ":9090",
    "enable_https": false
}
```

## Соответствие полей

| JSON поле | Переменная окружения | Флаг | Описание |
|-----------|---------------------|------|----------|
| `server_address` | `SERVER_ADDRESS` | `-a` | Адрес запуска HTTP-сервера |
| `base_url` | `BASE_URL` | `-b` | Базовый URL для сокращенных ссылок |
| `file_storage_path` | `FILE_STORAGE_PATH` | `-f` | Путь к файлу для хранения URL |
| `database_dsn` | `DATABASE_DSN` | `-d` | Строка подключения к базе данных PostgreSQL |
| `enable_https` | `ENABLE_HTTPS` | `-s` | Включить HTTPS сервер |
| `tls_cert_file` | `TLS_CERT_FILE` | `-tls-cert` | Путь к файлу TLS сертификата |
| `tls_key_file` | `TLS_KEY_FILE` | `-tls-key` | Путь к файлу TLS приватного ключа |
| `secret_key` | `SECRET_KEY` | `-secret-key` | Секретный ключ для подписи кук |
| `batch_delete_max_workers` | `BATCH_DELETE_MAX_WORKERS` | `-batch-max-workers` | Максимальное количество воркеров для параллельного удаления |
| `batch_delete_batch_size` | `BATCH_DELETE_BATCH_SIZE` | `-batch-size` | Размер батча для обработки URL |
| `batch_delete_sequential_threshold` | `BATCH_DELETE_SEQUENTIAL_THRESHOLD` | `-batch-sequential-threshold` | Порог для переключения на последовательное удаление |

## Примеры использования

### 1. Простая конфигурация для разработки

**dev_config.json:**
```json
{
    "server_address": ":3000",
    "base_url": "http://localhost:3000",
    "enable_https": false
}
```

**Запуск:**
```bash
go run ./cmd/shortener -c dev_config.json
```

### 2. Production конфигурация с HTTPS

**prod_config.json:**
```json
{
    "server_address": ":443",
    "base_url": "https://myshortener.com",
    "enable_https": true,
    "tls_cert_file": "/etc/ssl/certs/myshortener.crt",
    "tls_key_file": "/etc/ssl/private/myshortener.key",
    "database_dsn": "postgres://user:password@localhost/shortener_db",
    "secret_key": "super-secret-production-key"
}
```

**Запуск:**
```bash
go run ./cmd/shortener -config prod_config.json
```

### 3. Демонстрация приоритета конфигурации

**config.json:**
```json
{
    "server_address": ":8888",
    "base_url": "http://json.config"
}
```

**Запуск с переопределением через переменную окружения:**
```bash
export SERVER_ADDRESS=":9999"
go run ./cmd/shortener -c config.json
# Результат: server_address будет ":9999" (из переменной окружения)
# base_url будет "http://json.config" (из JSON файла)
```

**Запуск с переопределением через флаг:**
```bash
go run ./cmd/shortener -c config.json -a :7777
# Результат: server_address будет ":7777" (из флага - наивысший приоритет)
```

### 4. Конфигурация через переменную окружения CONFIG

```bash
export CONFIG=my_config.json
export SERVER_ADDRESS=":5000"  # Переопределяет значение из JSON
go run ./cmd/shortener
```

## Валидация конфигурации

### Корректные значения

- **enable_https**: `true` или `false` (boolean)
- **server_address**: строка в формате `:port` или `host:port`
- **Числовые поля**: положительные целые числа

### Обработка ошибок

1. **Файл не найден**: Приложение продолжит работу с настройками по умолчанию
2. **Некорректный JSON**: Приложение выдаст ошибку и завершит работу
3. **Некорректные значения полей**: Приложение может работать некорректно

## Особенности реализации

### Указатели в JSONConfig

JSON структура использует указатели (`*string`, `*bool`, `*int`) для различения:
- **Не установлено** (`nil`) - значение не указано в JSON
- **Пустое значение** (например, `""` для строк) - явно установлено пустое значение

### Применение конфигурации

Значения из JSON файла применяются только если соответствующие поля имеют значения по умолчанию:

```go
// Значение из JSON применится только если поле еще не изменено
if c.ServerAddress == ":8080" && jsonConfig.ServerAddress != nil {
    c.ServerAddress = *jsonConfig.ServerAddress
}
```

## Troubleshooting

### Ошибка "no such file or directory"

```bash
# Проверьте путь к файлу
ls -la config.json

# Или используйте абсолютный путь
go run ./cmd/shortener -c /absolute/path/to/config.json
```

### Ошибка "invalid character"

```bash
# Проверьте корректность JSON
cat config.json | python -m json.tool
# или
jq . config.json
```

### Конфигурация не применяется

1. Проверьте приоритет: флаги > переменные окружения > JSON
2. Убедитесь, что поля в JSON имеют правильные имена
3. Проверьте типы данных (boolean для enable_https, числа для batch_* полей)

## Преимущества JSON конфигурации

1. **Централизованная конфигурация**: Все настройки в одном файле
2. **Версионирование**: JSON файлы можно добавлять в git
3. **Простота развертывания**: Один файл конфигурации для разных окружений
4. **Читаемость**: JSON формат понятен и легко редактируется
5. **Гибкость**: Возможность частичной конфигурации (указывать только нужные поля) 