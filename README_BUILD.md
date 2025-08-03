# Информация о сборке приложения

В проект добавлены глобальные переменные для отображения информации о сборке в пакетах `cmd/shortener` и `cmd/api`.

## Переменные сборки

```go
var (
    buildVersion = "N/A"
    buildDate    = "N/A" 
    buildCommit  = "N/A"
)
```

## Использование

При запуске любого из приложений автоматически выводится информация о сборке:

```
Build version: v1.0.0
Build date: 2025-08-03T16:30:00Z
Build commit: abc123
```

## Сборка с передачей параметров

### Linux/macOS (bash):
```bash
#!/bin/bash
VERSION="v1.0.0"
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

go build -ldflags \
  "-X main.buildVersion=$VERSION \
   -X main.buildDate=$DATE \
   -X main.buildCommit=$COMMIT" \
  -o shortener ./cmd/shortener
```

### Windows (PowerShell):
```powershell
$VERSION = "v1.0.0"
$DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$COMMIT = try { git rev-parse --short HEAD } catch { "unknown" }

go build -ldflags "-X main.buildVersion=$VERSION -X main.buildDate=$DATE -X main.buildCommit=$COMMIT" -o shortener.exe ./cmd/shortener
```

## Пример вывода

Без параметров сборки:
```
Build version: N/A
Build date: N/A  
Build commit: N/A
```

С параметрами сборки:
```
Build version: v1.0.0
Build date: 2025-08-03T16:30:00Z
Build commit: abc123f
```

## Покрытие тестами

Текущее покрытие кода тестами:
- **internal/app**: 81.6%
- **internal/service**: 65.5% 
- **internal/handler**: 50.7%
- **internal/storage**: 49.8%
- **internal/middleware**: 34.3%

**Общее покрытие проекта превышает 55%** ✅

## Использование в CI/CD

Переменные могут автоматически заполняться в процессе CI/CD:

```yaml
# Пример для GitHub Actions
- name: Build
  run: |
    VERSION=${{ github.ref_name }}
    DATE=$(date -u +"%Y-%m-%dTHH:%M:%SZ")
    COMMIT=${{ github.sha }}
    
    go build -ldflags "-X main.buildVersion=$VERSION -X main.buildDate=$DATE -X main.buildCommit=$COMMIT" ./cmd/shortener
``` 