-- +goose Up

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS courses (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id),
    title      TEXT        NOT NULL,
    goal_depth SMALLINT    NOT NULL DEFAULT 2 CHECK (goal_depth BETWEEN 1 AND 3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chapters (
    id         BIGSERIAL PRIMARY KEY,
    course_id  BIGINT      NOT NULL REFERENCES courses (id),
    number     TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    position   INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (course_id, position)
);

CREATE TABLE IF NOT EXISTS lessons (
    id                   BIGSERIAL PRIMARY KEY,
    chapter_id           BIGINT      NOT NULL REFERENCES chapters (id),
    position             INT         NOT NULL,
    title                TEXT        NOT NULL,
    objective            TEXT        NOT NULL DEFAULT '',
    status               TEXT        NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending','in_progress','done','needs_review')),
    compaction_status    TEXT        NOT NULL DEFAULT 'pending'
                             CHECK (compaction_status IN ('pending','done','failed')),
    compaction_attempts  INT         NOT NULL DEFAULT 0,
    notes                TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chapter_id, position)
);

CREATE TABLE IF NOT EXISTS lesson_tags (
    id        BIGSERIAL PRIMARY KEY,
    lesson_id BIGINT NOT NULL REFERENCES lessons (id),
    tag       TEXT   NOT NULL,
    UNIQUE (lesson_id, tag)
);

CREATE TABLE IF NOT EXISTS lesson_content (
    id         BIGSERIAL PRIMARY KEY,
    lesson_id  BIGINT      NOT NULL REFERENCES lessons (id),
    content    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS lesson_chats (
    id         BIGSERIAL PRIMARY KEY,
    lesson_id  BIGINT      NOT NULL REFERENCES lessons (id),
    role       TEXT        NOT NULL CHECK (role IN ('user','assistant')),
    text       TEXT        NOT NULL,
    position   INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS knowledge_state (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT      NOT NULL REFERENCES users (id),
    tag            TEXT        NOT NULL,
    confidence     INT         NOT NULL DEFAULT 0 CHECK (confidence BETWEEN 0 AND 100),
    last_tested_at TIMESTAMPTZ,
    UNIQUE (user_id, tag)
);

CREATE TABLE IF NOT EXISTS test_results (
    id        BIGSERIAL PRIMARY KEY,
    lesson_id BIGINT      NOT NULL REFERENCES lessons (id),
    score     INT         NOT NULL CHECK (score BETWEEN 0 AND 100),
    feedback  TEXT,
    taken_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id),
    token_hash TEXT        NOT NULL UNIQUE,
    csrf_token TEXT        NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_courses_user_id          ON courses (user_id);
CREATE INDEX IF NOT EXISTS idx_chapters_course_id       ON chapters (course_id);
CREATE INDEX IF NOT EXISTS idx_lessons_chapter_id       ON lessons (chapter_id);
CREATE INDEX IF NOT EXISTS idx_lesson_tags_lesson_id    ON lesson_tags (lesson_id);
CREATE INDEX IF NOT EXISTS idx_lesson_content_lesson_id ON lesson_content (lesson_id);
CREATE INDEX IF NOT EXISTS idx_lesson_chats_lesson_id   ON lesson_chats (lesson_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_user_id        ON knowledge_state (user_id);
CREATE INDEX IF NOT EXISTS idx_test_results_lesson_id   ON test_results (lesson_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id   ON refresh_tokens (user_id);

-- +goose Down

DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS test_results;
DROP TABLE IF EXISTS knowledge_state;
DROP TABLE IF EXISTS lesson_chats;
DROP TABLE IF EXISTS lesson_content;
DROP TABLE IF EXISTS lesson_tags;
DROP TABLE IF EXISTS lessons;
DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS users;
