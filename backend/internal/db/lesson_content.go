package db

import "context"

func CreateLessonContent(ctx context.Context, q Querier, lessonID int64, content string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO lesson_content (lesson_id, content) VALUES ($1, $2)`,
		lessonID, content,
	)
	return err
}

func GetLessonContent(ctx context.Context, q Querier, lessonID int64) (string, error) {
	var content string
	err := q.QueryRow(ctx,
		`SELECT content FROM lesson_content
		 WHERE lesson_id = $1 ORDER BY created_at DESC LIMIT 1`,
		lessonID,
	).Scan(&content)
	if err != nil {
		return "", err
	}
	return content, nil
}
