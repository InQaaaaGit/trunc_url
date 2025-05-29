#!/bin/bash

# Пример использования API для удаления URL
# Убедитесь, что сервер запущен на localhost:8080

BASE_URL="http://localhost:8080"

echo "=== Демонстрация API удаления URL ==="

# 1. Создаем несколько коротких URL
echo "1. Создание коротких URL..."

URL1=$(curl -s -X POST "$BASE_URL/" \
  -H "Content-Type: text/plain" \
  -d "https://example.com" \
  -c cookies.txt)
echo "Создан URL1: $URL1"

URL2=$(curl -s -X POST "$BASE_URL/api/shorten" \
  -H "Content-Type: application/json" \
  -b cookies.txt -c cookies.txt \
  -d '{"url":"https://google.com"}' | jq -r '.result')
echo "Создан URL2: $URL2"

URL3=$(curl -s -X POST "$BASE_URL/api/shorten" \
  -H "Content-Type: application/json" \
  -b cookies.txt -c cookies.txt \
  -d '{"url":"https://github.com"}' | jq -r '.result')
echo "Создан URL3: $URL3"

# 2. Получаем список URL пользователя
echo -e "\n2. Получение списка URL пользователя..."
curl -s -X GET "$BASE_URL/api/user/urls" \
  -b cookies.txt | jq '.'

# 3. Извлекаем короткие ID из URL для удаления
SHORT_ID1=$(echo "$URL1" | sed 's|.*/||')
SHORT_ID2=$(echo "$URL2" | sed 's|.*/||')

echo -e "\n3. Удаление URL: $SHORT_ID1 и $SHORT_ID2"

# 4. Удаляем URL
curl -s -X DELETE "$BASE_URL/api/user/urls" \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d "[\"$SHORT_ID1\", \"$SHORT_ID2\"]"

echo "Удаление инициировано (статус 202 Accepted)"

# 5. Ждем немного для завершения асинхронного удаления
echo -e "\n4. Ожидание завершения удаления..."
sleep 2

# 6. Проверяем список URL после удаления
echo -e "\n5. Список URL после удаления:"
curl -s -X GET "$BASE_URL/api/user/urls" \
  -b cookies.txt | jq '.'

# 7. Пытаемся получить доступ к удаленному URL
echo -e "\n6. Попытка доступа к удаленному URL:"
echo "Запрос к $URL1:"
curl -s -w "HTTP Status: %{http_code}\n" -o /dev/null "$URL1"

echo "Запрос к $URL2:"
curl -s -w "HTTP Status: %{http_code}\n" -o /dev/null "$URL2"

echo "Запрос к $URL3 (не удален):"
curl -s -w "HTTP Status: %{http_code}\n" -o /dev/null "$URL3"

# Очистка
rm -f cookies.txt

echo -e "\n=== Демонстрация завершена ===" 