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

-- Вставка начальных данных 
DO $$
DECLARE
    bp1 INTEGER;
    bp2 INTEGER;
    bp3 INTEGER;
    bp4 INTEGER;
    p1 INTEGER;
    p2 INTEGER;
    p3 INTEGER;
    p4 INTEGER;
    t1 INTEGER;
    t2 INTEGER;
    t3 INTEGER;
    t4 INTEGER;
    f1 INTEGER;
    f2 INTEGER;
    f3 INTEGER;
    f4 INTEGER;
    f5 INTEGER;
    f6 INTEGER;
    m1 INTEGER;
    m2 INTEGER;
    m3 INTEGER;
    m4 INTEGER;
BEGIN
    -- Вставка в base_profile 
    INSERT INTO base_profile (username, domain)
    VALUES ('alexey', 'a4mail.ru') RETURNING id INTO bp1;
    
    INSERT INTO base_profile (username, domain)
    VALUES ('antonina', 'a4mail.ru') RETURNING id INTO bp2;
    
    INSERT INTO base_profile (username, domain)
    VALUES ('andrey', 'a4mail.ru') RETURNING id INTO bp3;
    
    INSERT INTO base_profile (username, domain)
    VALUES ('anna', 'a4mail.ru') RETURNING id INTO bp4;
    
    -- Вставка в profile
    INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday, phone_number)
    VALUES (bp1, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Alexey', 'Gusev', 'Nikolaevich', 'Male', '2003-08-20', '+77777777777') RETURNING id INTO p1;
    
    INSERT INTO profile (base_profile_id, password_hash,  name, surname, patronymic, gender, birthday, phone_number)
    VALUES (bp2, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Antonina', 'Andreeva', 'Aleksandrovna', 'Female', '2003-10-17', '+79697045539') RETURNING id INTO p2;
    
    INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday, phone_number)
    VALUES (bp3, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Andrey', 'Vavilov', 'Nikolaevich', 'Male', '2003-08-20', '+79099099090') RETURNING id INTO p3;
    
    INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday, phone_number)
    VALUES (bp4, '$2a$10$4PcooWbEMRjvdk2cMFumO.ajWaAclawIljtlfu2.2f5/fV8LkgEZe', 'Anna', 'Mihonina', 'Aleksandrovna', 'Female', '2003-08-20', '+79099499090') RETURNING id INTO p4;
    
    -- Вставка в thread 
    INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t1;
    INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t2;
    INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t3;
    INSERT INTO thread (root_message_id) VALUES (NULL) RETURNING id INTO t4;

    -- Вставка в message
    INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
    VALUES ('Topic1 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp1, t1) RETURNING id INTO m1;
    
    INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
    VALUES ('Topic2 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp1, t2) RETURNING id INTO m2;
    
    INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
    VALUES ('Topic3 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp2, t3) RETURNING id INTO m3;
    
    INSERT INTO message (topic, text, sender_base_profile_id, thread_id)
    VALUES ('Topic4 Lorem ipsum.', 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.', bp2, t4) RETURNING id INTO m4;
    
    -- Обновление root_message_id в thread
    UPDATE thread SET root_message_id = m1 WHERE id = t1;
    UPDATE thread SET root_message_id = m2 WHERE id = t2;
    UPDATE thread SET root_message_id = m3 WHERE id = t3;
    UPDATE thread SET root_message_id = m4 WHERE id = t4;
    
    -- Вставка в folder 
    INSERT INTO folder (profile_id, folder_name, folder_type) 
    VALUES (p1, 'Inbox', 'inbox') RETURNING id INTO f1;

    INSERT INTO folder (profile_id, folder_name, folder_type) 
    VALUES (p1, 'Sent', 'sent') RETURNING id INTO f2;
    
    INSERT INTO folder (profile_id, folder_name, folder_type) 
    VALUES (p2, 'Inbox', 'inbox') RETURNING id INTO f3;

    INSERT INTO folder (profile_id, folder_name, folder_type) 
    VALUES (p2, 'Sent', 'sent') RETURNING id INTO f4;
    
    INSERT INTO folder (profile_id, folder_name, folder_type) 
    VALUES (p3, 'Inbox', 'inbox') RETURNING id INTO f5;
    
    INSERT INTO folder (profile_id, folder_name, folder_type) 
    VALUES (p4, 'Inbox', 'inbox') RETURNING id INTO f6;

    -- Вставка в folder_profile_message
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m1, f2);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m1, f3);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m2, f2);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m2, f5);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m3, f4);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m3, f5);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m3, f6);
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m4, f4); 
    INSERT INTO folder_profile_message (message_id, folder_id) VALUES (m4, f6);

    -- Вставка в profile_message
    INSERT INTO profile_message (profile_id, message_id) VALUES (p1, m1);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p2, m1);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p1, m2);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p3, m2);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p2, m3);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p3, m3);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p4, m3);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p2, m4);
    INSERT INTO profile_message (profile_id, message_id) VALUES (p4, m4);
    
    -- Вставка в settings
    INSERT INTO settings (profile_id) VALUES (p1);
    INSERT INTO settings (profile_id) VALUES (p2);
    INSERT INTO settings (profile_id) VALUES (p3);
    INSERT INTO settings (profile_id) VALUES (p4);
END $$;