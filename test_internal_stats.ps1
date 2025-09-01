# Тестовый скрипт для демонстрации эндпоинта внутренней статистики (PowerShell)

Write-Host "=== Тестирование эндпоинта /api/internal/stats ===" -ForegroundColor Green
Write-Host

# Запускаем сервер в фоне с доверенной подсетью
Write-Host "Запуск сервера с доверенной подсетью 192.168.1.0/24..." -ForegroundColor Yellow
$serverProcess = Start-Process -FilePath "./api.exe" -ArgumentList "-t", "192.168.1.0/24" -PassThru -WindowStyle Hidden

# Ждем запуска сервера
Start-Sleep -Seconds 2

Write-Host
Write-Host "1. Тест: Запрос из доверенной подсети (X-Real-IP: 192.168.1.100)" -ForegroundColor Cyan
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/internal/stats" -Headers @{"X-Real-IP" = "192.168.1.100"} -Method Get
    $response | ConvertTo-Json
} catch {
    Write-Host "Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host

Write-Host "2. Тест: Запрос из недоверенной подсети (X-Real-IP: 10.0.0.100)" -ForegroundColor Cyan
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/internal/stats" -Headers @{"X-Real-IP" = "10.0.0.100"} -Method Get
    $response | ConvertTo-Json
} catch {
    Write-Host "Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host

Write-Host "3. Тест: Запрос без заголовка X-Real-IP (используется RemoteAddr)" -ForegroundColor Cyan
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/internal/stats" -Method Get
    $response | ConvertTo-Json
} catch {
    Write-Host "Ошибка: $($_.Exception.Message)" -ForegroundColor Red
}
Write-Host

# Останавливаем сервер
Write-Host "Остановка сервера..." -ForegroundColor Yellow
Stop-Process -Id $serverProcess.Id -Force

Write-Host
Write-Host "=== Тест завершен ===" -ForegroundColor Green 