# PowerShell скрипт для генерации Go кода из proto файла

Write-Host "Генерация Go кода из proto файла..." -ForegroundColor Green

# Создаем директорию proto если её нет
if (!(Test-Path "proto")) {
    New-Item -ItemType Directory -Path "proto"
}

# Генерируем Go код из proto файла
protoc --go_out=. --go_opt=paths=source_relative `
       --go-grpc_out=. --go-grpc_opt=paths=source_relative `
       proto/url_service.proto

Write-Host "Генерация завершена!" -ForegroundColor Green 