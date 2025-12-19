-- Store external receivers for messages so they can be returned in API responses
CREATE TABLE IF NOT EXISTS external_message_receiver (
    message_id INTEGER NOT NULL REFERENCES message(id) ON DELETE CASCADE,
    email TEXT NOT NULL CHECK (length(email) BETWEEN 3 AND 320),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, email)
);
