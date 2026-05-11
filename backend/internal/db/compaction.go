package db

import (
	"context"
	"time"
)

const MaxCompactionAttempts = 5

// QueueCompaction marks a lesson for (re)compaction from scratch.
func QueueCompaction(ctx context.Context, q Querier, lessonID int64) error {
	_, err := q.Exec(ctx,
		`UPDATE lessons
		 SET compaction_status = 'pending', compaction_attempts = 0, compaction_lease_until = NULL
		 WHERE id = $1`,
		lessonID,
	)
	return err
}

// ClaimCompaction leases up to limit pending lessons. Other workers skip
// leased rows until the lease expires, so a crashed worker's jobs are
// picked up again later.
func ClaimCompaction(ctx context.Context, q Querier, limit int, lease time.Duration) ([]*Lesson, error) {
	return queryLessons(ctx, q,
		`UPDATE lessons AS l
		 SET compaction_lease_until = now() + make_interval(secs => $2)
		 WHERE l.id IN (
		     SELECT id FROM lessons
		     WHERE compaction_status = 'pending'
		       AND compaction_attempts < $3
		       AND (compaction_lease_until IS NULL OR compaction_lease_until < now())
		     ORDER BY created_at
		     LIMIT $1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING `+lessonColumns,
		limit, lease.Seconds(), MaxCompactionAttempts,
	)
}

func CompleteCompaction(ctx context.Context, q Querier, lessonID int64) error {
	_, err := q.Exec(ctx,
		`UPDATE lessons SET compaction_status = 'done', compaction_lease_until = NULL WHERE id = $1`,
		lessonID,
	)
	return err
}

// FailCompaction schedules a retry with a linear backoff, or gives up after
// MaxCompactionAttempts.
func FailCompaction(ctx context.Context, q Querier, lessonID int64) error {
	_, err := q.Exec(ctx,
		`UPDATE lessons SET
		     compaction_attempts = compaction_attempts + 1,
		     compaction_status = CASE WHEN compaction_attempts + 1 >= $2 THEN 'failed' ELSE 'pending' END,
		     compaction_lease_until = now() + make_interval(mins => compaction_attempts + 1)
		 WHERE id = $1`,
		lessonID, MaxCompactionAttempts,
	)
	return err
}
