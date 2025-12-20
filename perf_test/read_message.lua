-- Скрипт для чтения одного сообщения по ID
math.randomseed(os.time())

wrk.method = "GET"
wrk.headers["Content-Type"] = "application/json"

-- Предполагаем, что у нас есть сообщения с ID от 1 до 100000
request = function()
    local message_id = math.random(1, 100000)
    
    -- Формируем правильный URL согласно коду gateway
    -- В коде: mux.Handle("GET /messages/{message_id}", http.HandlerFunc(s.messagePageHandler))
    return wrk.format("GET", "/messages/" .. tostring(message_id))
end

init = function(args)
    print("Initializing load test for single message reading...")
end

