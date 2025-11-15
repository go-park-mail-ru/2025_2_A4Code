-- +migrate Up

CREATE TYPE appeal_status AS ENUM ('open', 'in progress', 'closed');

CREATE TABLE IF NOT EXISTS appeal (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    topic TEXT CHECK (LENGTH(topic) BETWEEN 1 AND 200),
    text TEXT,
    profile_id INTEGER NOT NULL REFERENCES profile(id) ON DELETE NO ACTION,
    status appeal_status NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Триггер для updated_at в appeal
CREATE TRIGGER appeal_update_trigger
BEFORE UPDATE ON appeal
FOR EACH ROW EXECUTE PROCEDURE update_updated_at();