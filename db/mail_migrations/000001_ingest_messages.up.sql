CREATE TABLE IF NOT EXISTS ingest_messages (
    id BIGSERIAL PRIMARY KEY,
    mail_from   TEXT,
    rcpt_to     TEXT,
    subject     TEXT,
    raw_path    TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
