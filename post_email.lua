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

-- Основная функция построения запроса
request = function()
    -- Генерируем случайную строку для темы и текста (они должны быть одинаковыми)
    local random_content = get_random_string(10)
    
    -- Формируем JSON тело
    -- Примечание: thread_id добавлен по вашему требованию, 
    -- хотя в proto-файле на скриншоте его нет.
    local body = string.format(
        '{"topic": "%s", "text": "%s", "thread_id": "fixed_thread_1", "receivers": [{"email": "test_user@example.com"}], "files": []}',
        random_content,
        random_content
    )

    return wrk.format(nil, nil, nil, body)
end