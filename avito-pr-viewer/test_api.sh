#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"

echo -e "${BLUE}=== Тестирование API PR Reviewer Service ===${NC}\n"

# 1. Health check
echo -e "${BLUE}1. Health Check${NC}"
curl -s "$BASE_URL/health" | jq .
echo -e "\n"

# 2. Создание команды
echo -e "${BLUE}2. Создание команды 'backend'${NC}"
curl -s -X POST "$BASE_URL/team/add" \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }' | jq .
echo -e "\n"

# 3. Получение команды
echo -e "${BLUE}3. Получение команды 'backend'${NC}"
curl -s "$BASE_URL/team/get?team_name=backend" | jq .
echo -e "\n"

# 4. Создание PR
echo -e "${BLUE}4. Создание Pull Request${NC}"
curl -s -X POST "$BASE_URL/pullRequest/create" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "pull_request_name": "Add search feature",
    "author_id": "u1"
  }' | jq .
echo -e "\n"

# 5. Получение PR'ов пользователя u2 (должен быть ревьювером)
echo -e "${BLUE}5. Получение PR'ов пользователя u2 (ревьювера)${NC}"
curl -s "$BASE_URL/users/getReview?user_id=u2" | jq .
echo -e "\n"

# 6. Переназначение ревьювера
echo -e "${BLUE}6. Переназначение ревьювера${NC}"
curl -s -X POST "$BASE_URL/pullRequest/reassign" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u2"
  }' | jq .
echo -e "\n"

# 7. Установка активности пользователя
echo -e "${BLUE}7. Установка активности пользователя u3${NC}"
curl -s -X POST "$BASE_URL/users/setIsActive" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "u3",
    "is_active": false
  }' | jq .
echo -e "\n"

# 8. Merge PR
echo -e "${BLUE}8. Merge Pull Request${NC}"
curl -s -X POST "$BASE_URL/pullRequest/merge" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }' | jq .
echo -e "\n"

# 9. Попытка переназначить ревьювера у merged PR (должна быть ошибка)
echo -e "${BLUE}9. Попытка переназначить ревьювера у merged PR (ожидается ошибка)${NC}"
curl -s -X POST "$BASE_URL/pullRequest/reassign" \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001",
    "old_user_id": "u3"
  }' | jq .
echo -e "\n"

echo -e "${GREEN}=== Тестирование завершено ===${NC}"

