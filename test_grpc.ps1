# Скрипт для тестирования gRPC функциональности сервиса сокращения URL
# Использует HTTP адаптер для тестирования gRPC методов

param(
    [string]$ServerUrl = "http://localhost:8080"
)

# Конфигурация
$GRPC_ADAPTER_URL = "$ServerUrl/grpc"
$TempFile = [System.IO.Path]::GetTempFileName()

# Функции для вывода
function Write-Header {
    param([string]$Message)
    Write-Host "=== $Message ===" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Yellow
}

# Проверка доступности сервера
function Test-ServerAvailability {
    Write-Header "Проверка доступности сервера"
    
    try {
        $response = Invoke-RestMethod -Uri "$ServerUrl/ping" -Method Get -TimeoutSec 5
        Write-Success "Сервер доступен"
        return $true
    }
    catch {
        Write-Error "Сервер недоступен. Убедитесь, что сервер запущен с gRPC поддержкой"
        return $false
    }
}

# Тест создания короткого URL
function Test-CreateUrl {
    Write-Header "Тест создания короткого URL"
    
    $originalUrl = "https://example.com/test-grpc"
    
    try {
        $response = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/create?url=$originalUrl" -Method Post
        $shortUrl = $response.short_url
        Write-Success "URL создан: $shortUrl"
        $shortUrl | Out-File -FilePath $TempFile -Encoding UTF8
        return $true
    }
    catch {
        Write-Error "Ошибка создания URL: $($_.Exception.Message)"
        return $false
    }
}

# Тест получения оригинального URL
function Test-GetUrl {
    Write-Header "Тест получения оригинального URL"
    
    if (-not (Test-Path $TempFile)) {
        Write-Error "Файл с коротким URL не найден"
        return $false
    }
    
    $shortUrl = Get-Content $TempFile
    $shortId = [System.IO.Path]::GetFileName($shortUrl)
    
    try {
        $response = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/get?id=$shortId" -Method Get
        $originalUrl = $response.original_url
        Write-Success "Оригинальный URL получен: $originalUrl"
        return $true
    }
    catch {
        Write-Error "Ошибка получения URL: $($_.Exception.Message)"
        return $false
    }
}

# Тест ping
function Test-Ping {
    Write-Header "Тест ping"
    
    try {
        $response = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/ping" -Method Get
        Write-Success "Ping успешен"
        return $true
    }
    catch {
        Write-Error "Ошибка ping: $($_.Exception.Message)"
        return $false
    }
}

# Тест статистики
function Test-Stats {
    Write-Header "Тест получения статистики"
    
    try {
        $response = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/stats" -Method Get
        $urlsCount = $response.urls
        $usersCount = $response.users
        Write-Success "Статистика получена: URLs=$urlsCount, Users=$usersCount"
        return $true
    }
    catch {
        Write-Error "Ошибка получения статистики: $($_.Exception.Message)"
        return $false
    }
}

# Тест обработки ошибок
function Test-ErrorHandling {
    Write-Header "Тест обработки ошибок"
    
    # Тест пустого URL
    try {
        $response = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/create?url=" -Method Post
        Write-Error "Ошибка пустого URL не обработана"
        return $false
    }
    catch {
        Write-Success "Ошибка пустого URL обработана корректно"
    }
    
    # Тест несуществующего URL
    try {
        $response = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/get?id=nonexistent" -Method Get
        Write-Error "Ошибка несуществующего URL не обработана"
        return $false
    }
    catch {
        Write-Success "Ошибка несуществующего URL обработана корректно"
    }
    
    return $true
}

# Тест производительности
function Test-Performance {
    Write-Header "Тест производительности"
    
    $startTime = Get-Date
    
    # Создаем 10 URL
    for ($i = 1; $i -le 10; $i++) {
        try {
            $null = Invoke-RestMethod -Uri "$GRPC_ADAPTER_URL/create?url=https://example$i.com" -Method Post
        }
        catch {
            Write-Error "Ошибка создания URL ${i}: $($_.Exception.Message)"
            return $false
        }
    }
    
    $endTime = Get-Date
    $duration = ($endTime - $startTime).TotalSeconds
    
    Write-Success "Создано 10 URL за $duration секунд"
    return $true
}

# Очистка
function Clear-TempFiles {
    Write-Header "Очистка"
    
    if (Test-Path $TempFile) {
        Remove-Item $TempFile -Force
        Write-Success "Временные файлы удалены"
    }
}

# Основная функция
function Main {
    Write-Header "Тестирование gRPC функциональности"
    Write-Info "Используется HTTP адаптер для gRPC методов"
    
    $tests = @(
        @{ Name = "Server Availability"; Function = "Test-ServerAvailability" },
        @{ Name = "Ping"; Function = "Test-Ping" },
        @{ Name = "Create URL"; Function = "Test-CreateUrl" },
        @{ Name = "Get URL"; Function = "Test-GetUrl" },
        @{ Name = "Stats"; Function = "Test-Stats" },
        @{ Name = "Error Handling"; Function = "Test-ErrorHandling" },
        @{ Name = "Performance"; Function = "Test-Performance" }
    )
    
    $passed = 0
    $total = $tests.Count
    
    foreach ($test in $tests) {
        Write-Host ""
        $result = & $test.Function
        if ($result) {
            $passed++
        }
    }
    
    Write-Host ""
    Write-Header "Результаты тестирования"
    Write-Success "$passed из $total тестов прошли успешно"
    
    if ($passed -eq $total) {
        Write-Success "gRPC функциональность работает корректно"
    } else {
        Write-Error "Некоторые тесты не прошли"
        exit 1
    }
}

# Обработка завершения
try {
    Main
}
finally {
    Clear-TempFiles
} 