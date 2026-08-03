-- +goose Up
CREATE TABLE IF NOT EXISTS messages (
  chat_id VARCHAR(255) NOT NULL,
  message_id VARCHAR(255) NOT NULL,
  thread_id VARCHAR(255) NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  message_time TIMESTAMPTZ NOT NULL,
  chat_name VARCHAR(255) NOT NULL,
  chat_description TEXT,
  account_id VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  domain TEXT[],
  entities TEXT[],
  intent TEXT[],
  is_ai_handled BOOLEAN NOT NULL DEFAULT false,

  PRIMARY KEY (chat_id, message_id, thread_id)
);

-- +goose Down
SELECT 'down SQL query';
