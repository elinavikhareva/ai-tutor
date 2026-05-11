package db

import (
	"context"
	"time"
)

type TestResult struct {
	ID       int64     `json:"id"`
	LessonID int64     `json:"lesson_id"`
	Score    int       `json:"score"`
	Feedback string    `json:"feedback"`
	TakenAt  time.Time `json:"taken_at"`
}

func CreateTestResult(ctx context.Context, q Querier, lessonID int64, score int, feedback string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO test_results (lesson_id, score, feedback) VALUES ($1, $2, $3)`,
		lessonID, score, feedback,
	)
	return err
}

func GetLatestTestResult(ctx context.Context, q Querier, lessonID int64) (*TestResult, error) {
	r := &TestResult{}
	err := q.QueryRow(ctx,
		`SELECT id, lesson_id, score, feedback, taken_at
		 FROM test_results WHERE lesson_id = $1 ORDER BY taken_at DESC LIMIT 1`,
		lessonID,
	).Scan(&r.ID, &r.LessonID, &r.Score, &r.Feedback, &r.TakenAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}
