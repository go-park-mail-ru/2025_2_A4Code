\set app_password `echo $APP_DB_PASSWORD`

DO $$
BEGIN
    -- Часть 1: СОЗДАНИЕ ПОЛЬЗОВАТЕЛЯ
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'flintmail_app') THEN
        EXECUTE format('CREATE USER flintmail_app WITH 
            PASSWORD %L
            NOCREATEDB
            NOCREATEROLE
            NOINHERIT
            NOBYPASSRLS
            LOGIN
            CONNECTION LIMIT 50', :'app_password');
        
        RAISE NOTICE 'Создан сервисный пользователь flintmail_app';
    ELSE
        RAISE NOTICE 'Пользователь flintmail_app уже существует';
    END IF;
END
$$;

-- Часть 2: ПРАВА НА ТЕКУЩУЮ БАЗУ
GRANT CONNECT ON DATABASE a4code_db TO flintmail_app;
GRANT TEMPORARY ON DATABASE a4code_db TO flintmail_app;

-- Часть 3: ПРАВА НА СХЕМУ И ТАБЛИЦЫ
GRANT USAGE ON SCHEMA public TO flintmail_app;

-- base_profile - чтение и вставка
GRANT SELECT, INSERT ON base_profile TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE base_profile_id_seq TO flintmail_app;

-- profile - чтение, вставка, обновление
GRANT SELECT, INSERT, UPDATE ON profile TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE profile_id_seq TO flintmail_app;

-- settings - чтение и вставка
GRANT SELECT, INSERT ON settings TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE settings_id_seq TO flintmail_app;

-- folder - полные права
GRANT SELECT, INSERT, UPDATE, DELETE ON folder TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE folder_id_seq TO flintmail_app;

-- message - полные права
GRANT SELECT, INSERT, UPDATE, DELETE ON message TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE message_id_seq TO flintmail_app;

-- profile_message - чтение, вставка, обновление статусов
GRANT SELECT, INSERT, UPDATE ON profile_message TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE profile_message_id_seq TO flintmail_app;

-- folder_profile_message
GRANT SELECT, INSERT, DELETE ON folder_profile_message TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE folder_profile_message_id_seq TO flintmail_app;

-- file - чтение и вставка файлов
GRANT SELECT, INSERT ON file TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE file_id_seq TO flintmail_app;

-- thread - чтение, вставка, обновление
GRANT SELECT, INSERT, UPDATE ON thread TO flintmail_app;
GRANT USAGE, SELECT ON SEQUENCE thread_id_seq TO flintmail_app;

-- Часть 4: БЕЗОПАСНОСТЬ
REVOKE ALL ON DATABASE a4code_db FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM PUBLIC;

-- Часть 5: НАСТРОЙКИ
ALTER ROLE flintmail_app SET statement_timeout = '15s';
ALTER ROLE flintmail_app SET lock_timeout = '5s';

-- Часть 6: ДЕФОЛТНЫЕ ПРАВА ДЛЯ БУДУЩИХ ТАБЛИЦ
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO flintmail_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA public 
GRANT USAGE, SELECT ON SEQUENCES TO flintmail_app;

-- Часть 7: ЛОГИРОВАНИЕ
ALTER ROLE flintmail_app SET log_statement = 'ddl';

-- Часть 8: ФИНАЛЬНЫЙ ОТЧЕТ
DO $$
BEGIN
    RAISE NOTICE 'Пользователь flintmail_app настроен';
END
$$;