#!/bin/bash

# Параметры авторизации
LOGIN="janedoe"
PASSWORD="securepassword228"

# Константа для ID сообщения
MESSAGE_ID="4555"

echo "Logging in as $LOGIN..."

# 1. Получаем токен 
TOKEN=$(curl -s -X POST http://217.16.16.26:8000/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"login\":\"$LOGIN\",\"password\":\"$PASSWORD\"}" | \
  grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to get token. Check credentials or server status."
  exit 1
fi

echo "Token received: ${TOKEN:0:30}..."

# 2. Создаем временный Lua-скрипт для wrk
LUA_SCRIPT=$(mktemp)
cat <<EOF > "$LUA_SCRIPT"
wrk.method = "GET"
wrk.headers["Authorization"] = "Bearer $TOKEN"

request = function()
    local path = "/messages/$MESSAGE_ID"
    return wrk.format(nil, path)
end
EOF

echo "Starting load test for GET /messages/$MESSAGE_ID"

# 3. Запуск wrk
# Мы указываем базовый URL, а путь /messages/{id} добавится из Lua-скрипта
wrk -t4 -c20 -d4m -s "$LUA_SCRIPT" --latency "http://217.16.16.26:8000"

# 4. Удаление временного файла
rm "$LUA_SCRIPT"