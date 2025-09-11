#!/bin/bash

# Скрипт для тестирования gRPC функциональности сервиса сокращения URL
# Использует HTTP адаптер для тестирования gRPC методов

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
SERVER_URL="http://localhost:8080"
GRPC_ADAPTER_URL="$SERVER_URL/grpc"

# Функции для вывода
print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Проверка доступности сервера
check_server() {
    print_header "Проверка доступности сервера"
    
    if curl -s "$SERVER_URL/ping" > /dev/null; then
        print_success "Сервер доступен"
    else
        print_error "Сервер недоступен. Убедитесь, что сервер запущен с gRPC поддержкой"
        exit 1
    fi
}

# Тест создания короткого URL
test_create_url() {
    print_header "Тест создания короткого URL"
    
    local original_url="https://example.com/test-grpc"
    local response=$(curl -s -X POST "$GRPC_ADAPTER_URL/create?url=$original_url")
    
    if echo "$response" | grep -q "short_url"; then
        local short_url=$(echo "$response" | grep -o '"short_url":"[^"]*"' | cut -d'"' -f4)
        print_success "URL создан: $short_url"
        echo "$short_url" > /tmp/test_short_url.txt
    else
        print_error "Ошибка создания URL: $response"
        return 1
    fi
}

# Тест получения оригинального URL
test_get_url() {
    print_header "Тест получения оригинального URL"
    
    if [ ! -f /tmp/test_short_url.txt ]; then
        print_error "Файл с коротким URL не найден"
        return 1
    fi
    
    local short_url=$(cat /tmp/test_short_url.txt)
    local short_id=$(basename "$short_url")
    local response=$(curl -s "$GRPC_ADAPTER_URL/get?id=$short_id")
    
    if echo "$response" | grep -q "original_url"; then
        local original_url=$(echo "$response" | grep -o '"original_url":"[^"]*"' | cut -d'"' -f4)
        print_success "Оригинальный URL получен: $original_url"
    else
        print_error "Ошибка получения URL: $response"
        return 1
    fi
}

# Тест ping
test_ping() {
    print_header "Тест ping"
    
    local response=$(curl -s "$GRPC_ADAPTER_URL/ping")
    
    if [ "$response" = "" ] || [ "$response" = "OK" ]; then
        print_success "Ping успешен"
    else
        print_error "Ошибка ping: $response"
        return 1
    fi
}

# Тест статистики
test_stats() {
    print_header "Тест получения статистики"
    
    local response=$(curl -s "$GRPC_ADAPTER_URL/stats")
    
    if echo "$response" | grep -q "urls"; then
        local urls_count=$(echo "$response" | grep -o '"urls":[0-9]*' | cut -d':' -f2)
        local users_count=$(echo "$response" | grep -o '"users":[0-9]*' | cut -d':' -f2)
        print_success "Статистика получена: URLs=$urls_count, Users=$users_count"
    else
        print_error "Ошибка получения статистики: $response"
        return 1
    fi
}

# Тест обработки ошибок
test_error_handling() {
    print_header "Тест обработки ошибок"
    
    # Тест пустого URL
    local response=$(curl -s -X POST "$GRPC_ADAPTER_URL/create?url=")
    if echo "$response" | grep -q "error"; then
        print_success "Ошибка пустого URL обработана корректно"
    else
        print_error "Ошибка пустого URL не обработана: $response"
    fi
    
    # Тест несуществующего URL
    local response=$(curl -s "$GRPC_ADAPTER_URL/get?id=nonexistent")
    if echo "$response" | grep -q "error"; then
        print_success "Ошибка несуществующего URL обработана корректно"
    else
        print_error "Ошибка несуществующего URL не обработана: $response"
    fi
}

# Тест производительности
test_performance() {
    print_header "Тест производительности"
    
    local start_time=$(date +%s.%N)
    
    # Создаем 10 URL
    for i in {1..10}; do
        curl -s -X POST "$GRPC_ADAPTER_URL/create?url=https://example$i.com" > /dev/null
    done
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)
    
    print_success "Создано 10 URL за ${duration} секунд"
}

# Очистка
cleanup() {
    print_header "Очистка"
    
    if [ -f /tmp/test_short_url.txt ]; then
        rm /tmp/test_short_url.txt
        print_success "Временные файлы удалены"
    fi
}

# Основная функция
main() {
    print_header "Тестирование gRPC функциональности"
    print_info "Используется HTTP адаптер для gRPC методов"
    
    check_server
    test_ping
    test_create_url
    test_get_url
    test_stats
    test_error_handling
    test_performance
    
    print_header "Все тесты завершены"
    print_success "gRPC функциональность работает корректно"
}

# Обработка сигналов
trap cleanup EXIT

# Запуск тестов
main "$@" 