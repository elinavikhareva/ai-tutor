package db

import (
	"context"
	"time"
)

type Course struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"-"`
	Title     string    `json:"title"`
	GoalDepth int       `json:"goal_depth"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateCourse(ctx context.Context, q Querier, userID int64, title string, goalDepth int) (*Course, error) {
	c := &Course{}
	err := q.QueryRow(ctx,
		`INSERT INTO courses (user_id, title, goal_depth)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, goal_depth, created_at`,
		userID, title, goalDepth,
	).Scan(&c.ID, &c.UserID, &c.Title, &c.GoalDepth, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func GetCourse(ctx context.Context, q Querier, id, userID int64) (*Course, error) {
	c := &Course{}
	err := q.QueryRow(ctx,
		`SELECT id, user_id, title, goal_depth, created_at
		 FROM courses WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&c.ID, &c.UserID, &c.Title, &c.GoalDepth, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func ListCourses(ctx context.Context, q Querier, userID int64) ([]*Course, error) {
	rows, err := q.Query(ctx,
		`SELECT id, user_id, title, goal_depth, created_at
		 FROM courses WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courses := []*Course{}
	for rows.Next() {
		c := &Course{}
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.GoalDepth, &c.CreatedAt); err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}
