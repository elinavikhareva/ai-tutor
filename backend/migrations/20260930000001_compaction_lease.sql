-- +goose Up
ALTER TABLE lessons ADD COLUMN IF NOT EXISTS compaction_lease_until TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_lessons_compaction_pending
    ON lessons (created_at) WHERE compaction_status = 'pending';

-- Completing a lesson twice used to append the transcript again.
DELETE FROM lesson_chats a
    USING lesson_chats b
    WHERE a.lesson_id = b.lesson_id AND a.position = b.position AND a.id < b.id;

ALTER TABLE lesson_chats
    ADD CONSTRAINT lesson_chats_lesson_id_position_key UNIQUE (lesson_id, position);

-- +goose Down
ALTER TABLE lesson_chats DROP CONSTRAINT IF EXISTS lesson_chats_lesson_id_position_key;
DROP INDEX IF EXISTS idx_lessons_compaction_pending;
ALTER TABLE lessons DROP COLUMN IF EXISTS compaction_lease_until;
