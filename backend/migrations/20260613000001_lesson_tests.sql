-- +goose Up
CREATE TABLE IF NOT EXISTS lesson_tests (
    id         BIGSERIAL PRIMARY KEY,
    lesson_id  BIGINT      NOT NULL REFERENCES lessons (id),
    questions  JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_lesson_tests_lesson_id ON lesson_tests (lesson_id);

-- +goose Down
DROP TABLE IF EXISTS lesson_tests;
