package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func AddLessonTags(ctx context.Context, q Querier, lessonID int64, tags []string) error {
	_, err := q.Exec(ctx,
		`INSERT INTO lesson_tags (lesson_id, tag)
		 SELECT $1, unnest($2::text[])
		 ON CONFLICT (lesson_id, tag) DO NOTHING`,
		lessonID, tags,
	)
	return err
}

func ListLessonTags(ctx context.Context, q Querier, lessonID int64) ([]string, error) {
	rows, err := q.Query(ctx,
		`SELECT tag FROM lesson_tags WHERE lesson_id = $1 ORDER BY tag`,
		lessonID,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
