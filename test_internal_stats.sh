#!/bin/bash

# Тестовый скрипт для демонстрации эндпоинта внутренней статистики

echo "=== Тестирование эндпоинта /api/internal/stats ==="
echo

# Запускаем сервер в фоне с доверенной подсетью
echo "Запуск сервера с доверенной подсетью 192.168.1.0/24..."
./api -t "192.168.1.0/24" &
SERVER_PID=$!

# Ждем запуска сервера
sleep 2

echo
echo "1. Тест: Запрос из доверенной подсети (X-Real-IP: 192.168.1.100)"
curl -s -H "X-Real-IP: 192.168.1.100" http://localhost:8080/api/internal/stats
echo
echo

echo "2. Тест: Запрос из недоверенной подсети (X-Real-IP: 10.0.0.100)"
curl -s -H "X-Real-IP: 10.0.0.100" http://localhost:8080/api/internal/stats
echo
echo

echo "3. Тест: Запрос без заголовка X-Real-IP (используется RemoteAddr)"
curl -s http://localhost:8080/api/internal/stats
echo
echo

# Останавливаем сервер
echo "Остановка сервера..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo
echo "=== Тест завершен ===" 