-- Инициализация генератора случайных чисел
math.randomseed(os.time())

wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

-- Функция генерации случайной строки
local function get_random_string(length)
    local chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
    local str = ''
    for i = 1, length do
        local rand = math.random(#chars)
        str = str .. string.sub(chars, rand, rand)
    end
    return str
end

-- Счетчик для уникальных сообщений
local counter = 0

-- Основная функция построения запроса
request = function()
    counter = counter + 1
    
    -- Генерируем случайные данные
    local random_content = get_random_string(10)
    local thread_num = math.random(1, 100)
    local user_num = math.random(1, 1000)
    
    -- Формируем JSON тело
    local body = string.format(
        '{"topic": "Test %d - %s", "text": "Message text for test %d. Content: %s", "thread_id": "thread_%d", "receivers": [{"email": "user%d@example.com"}], "files": []}',
        counter,
        random_content,
        counter,
        random_content,
        thread_num,
        user_num
    )

    -- Устанавливаем длину контента
    wrk.headers["Content-Length"] = string.len(body)
    
    return wrk.format(nil, nil, nil, body)
end

-- Функция инициализации для каждого потока
init = function(args)
    print("Initializing load test for email creation...")
end
