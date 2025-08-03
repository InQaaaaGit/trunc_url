#!/bin/bash

# Скрипт для генерации самоподписанных TLS сертификатов для разработки

echo "Генерация самоподписанных TLS сертификатов для разработки..."

# Генерация приватного ключа
openssl genrsa -out server.key 2048

# Генерация сертификата на 365 дней
openssl req -new -x509 -key server.key -out server.crt -days 365 -subj "/C=RU/ST=Moscow/L=Moscow/O=TruncURL/OU=Dev/CN=localhost"

echo "Сертификаты созданы:"
echo "  - server.key (приватный ключ)"
echo "  - server.crt (сертификат)"
echo ""
echo "Для запуска HTTPS сервера используйте:"
echo "  go run ./cmd/shortener -s true"
echo "или установите переменную окружения:"
echo "  export ENABLE_HTTPS=true"
echo "  go run ./cmd/shortener" 