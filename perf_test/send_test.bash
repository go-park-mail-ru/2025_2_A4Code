#!/bin/bash

LOGIN="janedoe"
PASSWORD="securepassword228"
TARGET_EMAIL="janedoe@flintmail.ru"

echo "Logging in as $LOGIN..."

# Получаем токен
TOKEN=$(curl -s -X POST http://217.16.16.26:8000/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"login\":\"$LOGIN\",\"password\":\"$PASSWORD\"}" | \
  grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Error: Failed to get token. Check credentials or server status." [cite: 2]
  exit 1 [cite: 3]
fi

echo "Token received: ${TOKEN:0:30}..."

# Lua-скрипт 
LUA_SCRIPT=$(mktemp)
cat <<EOF > "$LUA_SCRIPT"
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Authorization"] = "Bearer $TOKEN"

request = function()
    local body = '{"topic":"Test","text":"Hello from load test","receivers":[{"email":"$TARGET_EMAIL"}],"files":[]}'
    return wrk.format(nil, nil, nil, body)
end
EOF

# Запуск wrk
wrk -t4 -c20 -d18m -s "$LUA_SCRIPT" --latency http://217.16.16.26:8000/messages/send

rm "$LUA_SCRIPT"