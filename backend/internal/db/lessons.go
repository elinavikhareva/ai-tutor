package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	LessonPending     = "pending"
	LessonInProgress  = "in_progress"
	LessonDone        = "done"
	LessonNeedsReview = "needs_review"
)

func ValidLessonStatus(s string) bool {
	switch s {
	case LessonPending, LessonInProgress, LessonDone, LessonNeedsReview:
		return true
	}
	return false
}

type Lesson struct {
	ID                 int64     `json:"id"`
	ChapterID          int64     `json:"chapter_id"`
	Position           int       `json:"position"`
	Title              string    `json:"title"`
	Objective          string    `json:"objective"`
	Status             string    `json:"status"`
	CompactionStatus   string    `json:"compaction_status"`
	CompactionAttempts int       `json:"-"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

const lessonColumns = `l.id, l.chapter_id, l.position, l.title, l.objective,
	l.status, l.compaction_status, l.compaction_attempts, l.notes, l.created_at`

func scanLesson(row pgx.Row) (*Lesson, error) {
	l := &Lesson{}
	err := row.Scan(
		&l.ID, &l.ChapterID, &l.Position, &l.Title, &l.Objective,
		&l.Status, &l.CompactionStatus, &l.CompactionAttempts, &l.Notes, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func queryLessons(ctx context.Context, q Querier, sql string, args ...any) ([]*Lesson, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lessons []*Lesson
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		lessons = append(lessons, l)
	}
	return lessons, rows.Err()
}

func CreateLesson(ctx context.Context, q Querier, chapterID int64, position int, title, objective string) (*Lesson, error) {
	return scanLesson(q.QueryRow(ctx,
		`INSERT INTO lessons AS l (chapter_id, position, title, objective)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+lessonColumns,
		chapterID, position, title, objective,
	))
}

func GetLesson(ctx context.Context, q Querier, id int64) (*Lesson, error) {
	return scanLesson(q.QueryRow(ctx,
		`SELECT `+lessonColumns+` FROM lessons l WHERE l.id = $1`,
		id,
	))
}

// GetLessonForUser returns ErrNotFound if the lesson belongs to another user.
func GetLessonForUser(ctx context.Context, q Querier, lessonID, userID int64) (*Lesson, error) {
	return scanLesson(q.QueryRow(ctx,
		`SELECT `+lessonColumns+`
		 FROM lessons l
		 JOIN chapters ch ON ch.id = l.chapter_id
		 JOIN courses  c  ON c.id  = ch.course_id
		 WHERE l.id = $1 AND c.user_id = $2`,
		lessonID, userID,
	))
}

func ListCourseLessons(ctx context.Context, q Querier, courseID int64) ([]*Lesson, error) {
	return queryLessons(ctx, q,
		`SELECT `+lessonColumns+`
		 FROM lessons l
		 JOIN chapters ch ON ch.id = l.chapter_id
		 WHERE ch.course_id = $1
		 ORDER BY ch.position, l.position`,
		courseID,
	)
}

func UpdateLessonStatus(ctx context.Context, q Querier, id int64, status string) error {
	_, err := q.Exec(ctx, `UPDATE lessons SET status = $1 WHERE id = $2`, status, id)
	return err
}

func UpdateLessonNotes(ctx context.Context, q Querier, id int64, notes string) error {
	_, err := q.Exec(ctx, `UPDATE lessons SET notes = $1 WHERE id = $2`, notes, id)
	return err
}

func GetLessonGoalDepth(ctx context.Context, q Querier, lessonID int64) (int, error) {
	var depth int
	err := q.QueryRow(ctx,
		`SELECT c.goal_depth
		 FROM lessons l
		 JOIN chapters ch ON ch.id = l.chapter_id
		 JOIN courses  c  ON c.id  = ch.course_id
		 WHERE l.id = $1`,
		lessonID,
	).Scan(&depth)
	return depth, err
}
