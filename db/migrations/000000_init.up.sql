-- +migrate Up
-- Создание таблицы базовых профилей (base_profile)
CREATE TABLE IF NOT EXISTS base_profile (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    username TEXT NOT NULL CHECK (LENGTH(username) BETWEEN 1 AND 50),
    domain TEXT NOT NULL CHECK (LENGTH(domain) BETWEEN 1 AND 50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (username),
    UNIQUE (username, domain)
);    

-- Триггер для updated_at в base_profile
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER base_profile_update_trigger
BEFORE UPDATE ON base_profile
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Создание таблицы профилей (profile)
CREATE TABLE IF NOT EXISTS profile (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    base_profile_id INTEGER NOT NULL UNIQUE REFERENCES base_profile(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL CHECK (LENGTH(password_hash) <= 255),
    name TEXT CHECK (LENGTH(name) BETWEEN 1 AND 50),  
    surname TEXT CHECK (LENGTH(surname) BETWEEN 1 AND 200),
    patronymic TEXT CHECK (LENGTH(patronymic) BETWEEN 1 AND 200),
    gender TEXT CHECK (gender IN ('Male', 'Female')),
    birthday DATE,
    image_path TEXT CHECK (LENGTH(image_path) BETWEEN 1 AND 200),
    phone_number TEXT UNIQUE CHECK (LENGTH(phone_number) BETWEEN 1 AND 20),
    auth_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в profile
CREATE TRIGGER profile_update_trigger
BEFORE UPDATE ON profile
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE TABLE refresh_tokens (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    token TEXT NOT NULL UNIQUE CHECK (LENGTH(token) BETWEEN 32 AND 64),
    profile_id INTEGER NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    user_agent TEXT CHECK (LENGTH(user_agent) <= 500),
    ip_address INET,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER refresh_tokens_update_trigger
BEFORE UPDATE ON refresh_tokens
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Создание таблицы цепочек сообщений (thread) без foreign key сначала
CREATE TABLE IF NOT EXISTS thread (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    root_message_id INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в thread
CREATE TRIGGER thread_update_trigger
BEFORE UPDATE ON thread
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Создание таблицы сообщений (message)
-- DateOfDispatch - бизнес-время отправки (может отличаться от created_at, напр. для черновиков)
CREATE TABLE IF NOT EXISTS message (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    topic TEXT CHECK (LENGTH(topic) BETWEEN 1 AND 200),
    text TEXT,
    date_of_dispatch TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sender_base_profile_id INTEGER NOT NULL REFERENCES base_profile(id) ON DELETE NO ACTION,
    thread_id INTEGER REFERENCES thread(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в message
CREATE TRIGGER message_update_trigger
BEFORE UPDATE ON message
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Добавление foreign key в thread после создания message
ALTER TABLE thread
ADD CONSTRAINT fk_thread_root_message
FOREIGN KEY (root_message_id) REFERENCES message(id) ON DELETE SET NULL;

-- Создание таблицы папок (folder)
CREATE TABLE IF NOT EXISTS folder(
    id        INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	profile_id INTEGER NOT NULL REFERENCES profile(id) ON DELETE NO ACTION,
	folder_name      TEXT NOT NULL CHECK (LENGTH(folder_name) BETWEEN 1 AND 50),
	folder_type      TEXT NOT NULL CHECK (folder_type IN ('inbox', 'sent', 'draft', 'spam', 'trash', 'custom')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в folder
CREATE TRIGGER folder_update_trigger
BEFORE UPDATE ON folder
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();


-- Создание таблицы файлов (file)
CREATE TABLE IF NOT EXISTS file (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    file_type TEXT NOT NULL CHECK (LENGTH(file_type) BETWEEN 1 AND 100),
    size BIGINT NOT NULL CHECK (size >= 0 AND size <= 1073741824),                       -- 1 GB максимум
        CONSTRAINT file_size_limit CHECK (
        (file_type IN ('image', 'avatar') AND size <= 10485760) OR                       -- 10 MB для изображений
        (file_type = 'document' AND size <= 52428800) OR                                 -- 50 MB для документов  
        (file_type = 'video' AND size <= 1073741824) OR                                  -- 1 GB для видео
        (file_type NOT IN ('image', 'avatar', 'document', 'video') AND size <= 10485760) -- 10 MB по умолчанию
        ),
    storage_path TEXT NOT NULL CHECK (
        LENGTH(storage_path) BETWEEN 1 AND 200 AND
        storage_path ~ '^[\w\-./]+[\w\-]$' AND       -- разрешенные символы, не заканчивается на слеш/точку
        storage_path !~ '\.\.' AND                   -- запрет на parent directory traversal
        storage_path !~ '^/' AND                     -- относительные пути
        storage_path ~ '\.\w{1,10}$'                 -- должно быть расширение файла (1-10 символов)
    ),
    message_id INTEGER NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в file
CREATE TRIGGER file_update_trigger
BEFORE UPDATE ON file
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Создание таблицы связи профилей и сообщений и папок (folder_profile_message)
CREATE TABLE IF NOT EXISTS folder_profile_message (
    message_id INTEGER NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    folder_id INTEGER NOT NULL REFERENCES folder(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, folder_id)
);

-- Триггер для updated_at в folder_profile_message
CREATE TRIGGER folder_profile_message_update_trigger
BEFORE UPDATE ON folder_profile_message
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Создание таблицы связи профилей и сообщений (profile_message)
CREATE TABLE IF NOT EXISTS profile_message (
    profile_id INTEGER NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    message_id INTEGER NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    read_status BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (profile_id, message_id)
);

-- Триггер для updated_at в profile_message
CREATE TRIGGER profile_message_update_trigger
BEFORE UPDATE ON profile_message
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

-- Создание таблицы настроек (settings)
CREATE TYPE app_language AS ENUM ('ru', 'en', 'de', 'fr', 'es', 'zh', 'ja');

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    profile_id INTEGER NOT NULL UNIQUE REFERENCES profile(id) ON DELETE CASCADE,
    notification_tolerance INTEGER NOT NULL DEFAULT 0 CHECK (notification_tolerance >= 0),
    language app_language NOT NULL DEFAULT 'ru',
    theme TEXT NOT NULL DEFAULT 'light' CHECK (LENGTH(theme) BETWEEN 1 AND 50),
    signature TEXT CHECK (LENGTH(signature) <= 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в settings
CREATE TRIGGER settings_update_trigger
BEFORE UPDATE ON settings
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

