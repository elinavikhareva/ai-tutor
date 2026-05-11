package db

import "context"

func CreateLessonTest(ctx context.Context, q Querier, lessonID int64, questions string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO lesson_tests (lesson_id, questions) VALUES ($1, $2)`,
		lessonID, questions,
	)
	return err
}

func GetLatestLessonTest(ctx context.Context, q Querier, lessonID int64) (string, error) {
	var questions string
	err := q.QueryRow(ctx,
		`SELECT questions FROM lesson_tests
		 WHERE lesson_id = $1 ORDER BY created_at DESC LIMIT 1`,
		lessonID,
	).Scan(&questions)
	if err != nil {
		return "", err
	}
	return questions, nil
}
