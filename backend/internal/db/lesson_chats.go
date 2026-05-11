package db

import (
	"context"
	"time"
)

type ChatMessage struct {
	ID        int64
	LessonID  int64
	Role      string
	Text      string
	Position  int
	CreatedAt time.Time
}

// ReplaceChatMessages stores the full transcript of a lesson, dropping any
// previously saved one.
func ReplaceChatMessages(ctx context.Context, q Querier, lessonID int64, msgs []ChatMessage) error {
	roles := make([]string, len(msgs))
	texts := make([]string, len(msgs))
	for i, m := range msgs {
		roles[i], texts[i] = m.Role, m.Text
	}
	if _, err := q.Exec(ctx, `DELETE FROM lesson_chats WHERE lesson_id = $1`, lessonID); err != nil {
		return err
	}
	_, err := q.Exec(ctx,
		`INSERT INTO lesson_chats (lesson_id, role, text, position)
		 SELECT $1, m.role, m.text, m.position
		 FROM unnest($2::text[], $3::text[]) WITH ORDINALITY AS m(role, text, position)`,
		lessonID, roles, texts,
	)
	return err
}

func ListChatMessages(ctx context.Context, q Querier, lessonID int64) ([]*ChatMessage, error) {
	rows, err := q.Query(ctx,
		`SELECT id, lesson_id, role, text, position, created_at
		 FROM lesson_chats WHERE lesson_id = $1 ORDER BY position`,
		lessonID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*ChatMessage
	for rows.Next() {
		m := &ChatMessage{}
		if err := rows.Scan(&m.ID, &m.LessonID, &m.Role, &m.Text, &m.Position, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
