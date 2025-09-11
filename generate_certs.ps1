# PowerShell скрипт для генерации самоподписанных TLS сертификатов для разработки

Write-Host "Генерация самоподписанных TLS сертификатов для разработки..." -ForegroundColor Green

# Проверяем наличие OpenSSL
try {
    openssl version | Out-Null
    Write-Host "OpenSSL найден, используем OpenSSL..." -ForegroundColor Yellow
    
    # Генерация приватного ключа
    openssl genrsa -out server.key 2048
    
    # Генерация сертификата на 365 дней
    openssl req -new -x509 -key server.key -out server.crt -days 365 -subj "/C=RU/ST=Moscow/L=Moscow/O=TruncURL/OU=Dev/CN=localhost"
    
} catch {
    Write-Host "OpenSSL не найден, используем PowerShell для создания сертификата..." -ForegroundColor Yellow
    
    # Альтернативный способ создания сертификата через PowerShell
    $cert = New-SelfSignedCertificate -DnsName "localhost" -CertStoreLocation "cert:\LocalMachine\My" -KeyExportPolicy Exportable -KeySpec Signature -KeyLength 2048 -KeyAlgorithm RSA -HashAlgorithm SHA256 -Subject "CN=localhost,O=TruncURL,L=Moscow,S=Moscow,C=RU"
    
    # Экспорт сертификата
    $pwd = ConvertTo-SecureString -String "password" -Force -AsPlainText
    Export-PfxCertificate -Cert $cert -FilePath "server.pfx" -Password $pwd
    Export-Certificate -Cert $cert -FilePath "server.crt" -Type CERT
    
    # Извлечение приватного ключа (требует дополнительных шагов на Windows)
    Write-Host "Для извлечения приватного ключа используйте:" -ForegroundColor Cyan
    Write-Host "openssl pkcs12 -in server.pfx -nocerts -out server.key -password pass:password -nodes" -ForegroundColor Cyan
}

Write-Host ""
Write-Host "Сертификаты созданы:" -ForegroundColor Green
Write-Host "  - server.key (приватный ключ)" -ForegroundColor White
Write-Host "  - server.crt (сертификат)" -ForegroundColor White
Write-Host ""
Write-Host "Для запуска HTTPS сервера используйте:" -ForegroundColor Cyan
Write-Host "  go run ./cmd/shortener -s true" -ForegroundColor White
Write-Host "или установите переменную окружения:" -ForegroundColor Cyan
Write-Host "  `$env:ENABLE_HTTPS='true'" -ForegroundColor White
Write-Host "  go run ./cmd/shortener" -ForegroundColor White 