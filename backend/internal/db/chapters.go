package db

import (
	"context"
	"time"
)

type Chapter struct {
	ID        int64
	CourseID  int64
	Number    string
	Title     string
	Position  int
	CreatedAt time.Time
}

func CreateChapter(ctx context.Context, q Querier, courseID int64, number, title string, position int) (*Chapter, error) {
	c := &Chapter{}
	err := q.QueryRow(ctx,
		`INSERT INTO chapters (course_id, number, title, position)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, course_id, number, title, position, created_at`,
		courseID, number, title, position,
	).Scan(&c.ID, &c.CourseID, &c.Number, &c.Title, &c.Position, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func ListChapters(ctx context.Context, q Querier, courseID int64) ([]*Chapter, error) {
	rows, err := q.Query(ctx,
		`SELECT id, course_id, number, title, position, created_at
		 FROM chapters WHERE course_id = $1 ORDER BY position`,
		courseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []*Chapter
	for rows.Next() {
		c := &Chapter{}
		if err := rows.Scan(&c.ID, &c.CourseID, &c.Number, &c.Title, &c.Position, &c.CreatedAt); err != nil {
			return nil, err
		}
		chapters = append(chapters, c)
	}
	return chapters, rows.Err()
}
