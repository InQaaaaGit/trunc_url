#!/bin/bash

# Скрипт для генерации Go кода из proto файла

echo "Генерация Go кода из proto файла..."

# Создаем директорию proto если её нет
mkdir -p proto

# Генерируем Go код из proto файла
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/url_service.proto

echo "Генерация завершена!" 