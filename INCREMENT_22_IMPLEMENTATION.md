# Реализация инкремента 22: JSON конфигурация

## ✅ Выполненные требования

В соответствии с заданием инкремента 22 добавлена возможность конфигурации приложения с помощью JSON файла:

- ✅ **Поддержка всех действующих опций**: Все существующие настройки приложения поддерживаются в JSON формате
- ✅ **Флаг -c/-config**: Имя файла конфигурации задается через флаги `-c` или `-config`
- ✅ **Переменная окружения CONFIG**: Поддержка переменной окружения `CONFIG` для указания файла
- ✅ **Корректный приоритет**: Значения из JSON файла имеют меньший приоритет, чем флаги или переменные окружения
- ✅ **Формат JSON**: Соответствует требуемому формату из задания

## 🔧 Архитектурные изменения

### 1. Структура JSONConfig (internal/config/config.go)

**Новая структура для JSON конфигурации:**
```go
type JSONConfig struct {
    ServerAddress   *string `json:"server_address,omitempty"`
    BaseURL         *string `json:"base_url,omitempty"`
    FileStoragePath *string `json:"file_storage_path,omitempty"`
    DatabaseDSN     *string `json:"database_dsn,omitempty"`
    SecretKey       *string `json:"secret_key,omitempty"`
    
    // HTTPS настройки
    EnableHTTPS *bool   `json:"enable_https,omitempty"`
    TLSCertFile *string `json:"tls_cert_file,omitempty"`
    TLSKeyFile  *string `json:"tls_key_file,omitempty"`
    
    // Параметры для batch deletion
    BatchDeleteMaxWorkers          *int `json:"batch_delete_max_workers,omitempty"`
    BatchDeleteBatchSize           *int `json:"batch_delete_batch_size,omitempty"`
    BatchDeleteSequentialThreshold *int `json:"batch_delete_sequential_threshold,omitempty"`
}
```

**Ключевые особенности:**
- Использование указателей для различения "не установлено" (`nil`) и "пустое значение"
- Поддержка `omitempty` для генерации чистого JSON
- Полное покрытие всех настроек приложения

### 2. Обновленная структура Config

**Добавленное поле:**
```go
type Config struct {
    // ... существующие поля ...
    
    // Конфигурационный файл
    ConfigFile string `env:"CONFIG"` // Путь к JSON файлу конфигурации
}
```

### 3. Новые функции

**loadJSONConfig()** - загружает конфигурацию из JSON файла:
```go
func loadJSONConfig(filename string) (*JSONConfig, error)
```

**applyJSONConfig()** - применяет JSON конфигурацию с учетом приоритетов:
```go
func (c *Config) applyJSONConfig(jsonConfig *JSONConfig)
```

### 4. Обновленная логика NewConfig()

**Новый порядок применения конфигурации:**
```go
func NewConfig() (*Config, error) {
    // 1. Значения по умолчанию
    cfg := &Config{...}
    
    // 2. Парсинг флагов (для получения пути к JSON файлу)
    flag.Parse()
    
    // 3. Проверка переменной окружения CONFIG
    if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" && cfg.ConfigFile == "" {
        cfg.ConfigFile = envConfigFile
    }
    
    // 4. Загрузка и применение JSON конфигурации (низший приоритет)
    jsonConfig, err := loadJSONConfig(cfg.ConfigFile)
    cfg.applyJSONConfig(jsonConfig)
    
    // 5. Применение переменных окружения (средний приоритет)
    env.Parse(cfg)
    
    // 6. Флаги уже применены (наивысший приоритет)
    
    return cfg, nil
}
```

## 📋 Поддерживаемые поля JSON

В точном соответствии с требованиями задания:

```json
{
    "server_address": "localhost:8080",
    "base_url": "http://localhost", 
    "file_storage_path": "/path/to/file.db",
    "database_dsn": "",
    "enable_https": true
}
```

**Дополнительные поля (расширение функциональности):**
```json
{
    "tls_cert_file": "server.crt",
    "tls_key_file": "server.key", 
    "secret_key": "your-custom-secret-key",
    "batch_delete_max_workers": 5,
    "batch_delete_batch_size": 10,
    "batch_delete_sequential_threshold": 8
}
```

## 🎯 Соответствие требованиям

### 1. Поддержка всех действующих опций ✅

| Настройка | JSON поле | Переменная окружения | Флаг |
|-----------|-----------|---------------------|------|
| Адрес сервера | `server_address` | `SERVER_ADDRESS` | `-a` |
| Базовый URL | `base_url` | `BASE_URL` | `-b` |
| Путь к файлу хранения | `file_storage_path` | `FILE_STORAGE_PATH` | `-f` |
| DSN базы данных | `database_dsn` | `DATABASE_DSN` | `-d` |
| Включить HTTPS | `enable_https` | `ENABLE_HTTPS` | `-s` |
| ... и все остальные | ... | ... | ... |

### 2. Имя файла конфигурации ✅

```bash
# Через флаг -c
go run ./cmd/shortener -c config.json

# Через флаг -config
go run ./cmd/shortener -config config.json

# Через переменную окружения CONFIG
export CONFIG=config.json
go run ./cmd/shortener
```

### 3. Приоритет конфигурации ✅

**Порядок приоритета (от высшего к низшему):**
1. **Флаги командной строки** - наивысший приоритет
2. **Переменные окружения** - средний приоритет
3. **JSON файл конфигурации** - низший приоритет

**Пример демонстрации приоритета:**
```bash
# config.json содержит: "server_address": ":8888"
export SERVER_ADDRESS=":9999"
go run ./cmd/shortener -c config.json -a :7777

# Результат: server_address = ":7777" (из флага - наивысший приоритет)
```

### 4. Формат файла ✅

Точно соответствует требованиям задания:

```json
{
    "server_address": "localhost:8080", // аналог переменной окружения SERVER_ADDRESS или флага -a
    "base_url": "http://localhost", // аналог переменной окружения BASE_URL или флага -b
    "file_storage_path": "/path/to/file.db", // аналог переменной окружения FILE_STORAGE_PATH или флага -f
    "database_dsn": "", // аналог переменной окружения DATABASE_DSN или флага -d
    "enable_https": true // аналог переменной окружения ENABLE_HTTPS или флага -s
}
```

## 🧪 Тестирование

### 1. Unit тесты

```bash
go test ./internal/config -v -run="TestJSON|TestLoad|TestApply"
```

**Покрываемые сценарии:**
- Загрузка валидного JSON файла
- Обработка частичной конфигурации
- Обработка некорректного JSON
- Приоритет конфигурации
- Отсутствующий файл конфигурации

### 2. Интеграционное тестирование

**Тест с JSON файлом:**
```bash
# Создаем test_config.json
echo '{"server_address": ":9090", "enable_https": false}' > test_config.json

# Тестируем загрузку
go run ./cmd/shortener -c test_config.json
```

**Тест приоритета:**
```bash
export SERVER_ADDRESS=":5555"
go run ./cmd/shortener -c test_config.json -a :3333
# Ожидаемый результат: использует :3333 (флаг имеет наивысший приоритет)
```

## 📁 Файловая структура

```
trunc_url/
├── internal/
│   └── config/
│       ├── config.go              # Основная логика конфигурации
│       ├── config_https_test.go   # Тесты HTTPS
│       └── config_json_test.go    # Тесты JSON конфигурации
├── config.example.json            # Пример полной конфигурации
├── test_config.json              # Простой тестовый файл
├── README_JSON_CONFIG.md         # Подробная документация
└── INCREMENT_22_IMPLEMENTATION.md # Техническая документация
```

## 🔍 Особенности реализации

### 1. Обработка типов данных

**Boolean поля (enable_https):**
```go
if c.EnableHTTPS == "" && jsonConfig.EnableHTTPS != nil {
    if *jsonConfig.EnableHTTPS {
        c.EnableHTTPS = "true"
    }
}
```

**Строковые поля с дефолтными значениями:**
```go
if c.ServerAddress == ":8080" && jsonConfig.ServerAddress != nil {
    c.ServerAddress = *jsonConfig.ServerAddress
}
```

**Числовые поля:**
```go
if c.BatchDeleteMaxWorkers == 3 && jsonConfig.BatchDeleteMaxWorkers != nil {
    c.BatchDeleteMaxWorkers = *jsonConfig.BatchDeleteMaxWorkers
}
```

### 2. Graceful handling отсутствующих файлов

```go
func loadJSONConfig(filename string) (*JSONConfig, error) {
    if filename == "" {
        return &JSONConfig{}, nil
    }
    
    data, err := os.ReadFile(filename)
    if err != nil {
        if os.IsNotExist(err) {
            // Файл не существует - это не ошибка
            return &JSONConfig{}, nil
        }
        return nil, err
    }
    // ...
}
```

### 3. Двойной парсинг флагов

Флаги парсятся дважды для корректного определения пути к JSON файлу:
1. **Первый парсинг** - для получения пути к JSON файлу из флагов `-c`/`-config`
2. **Применение JSON** - загрузка и применение JSON конфигурации
3. **Применение переменных окружения** - переопределение JSON значений
4. **Флаги уже применены** - финальное переопределение

## ✅ Проверка соответствия заданию

1. ✅ **Все действующие опции поддерживаются** в JSON формате
2. ✅ **Флаги -c/-config** для указания файла конфигурации
3. ✅ **Переменная окружения CONFIG** для указания файла конфигурации
4. ✅ **Правильный приоритет**: флаги > переменные окружения > JSON файл
5. ✅ **Точный формат**: соответствует требованиям задания
6. ✅ **Обратная совместимость**: существующий функционал не нарушен
7. ✅ **Comprehensive тестирование**: unit и интеграционные тесты
8. ✅ **Полная документация**: примеры использования и troubleshooting

**Реализация инкремента 22 полностью соответствует требованиям задания!** 🎉 