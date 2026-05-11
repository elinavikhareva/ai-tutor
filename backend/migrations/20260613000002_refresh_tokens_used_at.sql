-- +goose Up
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS used_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS used_at;
