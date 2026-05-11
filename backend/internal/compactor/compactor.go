// Package compactor turns finished lesson chats into lecture notes in the
// background.
package compactor

import (
	"context"
	"log/slog"
	"time"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/metrics"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
)

type Compactor struct {
	db       *db.DB
	tutor    *tutor.Service
	log      *slog.Logger
	interval time.Duration
	batch    int
	// lease must outlast the slowest LLM round trip, otherwise another
	// worker may pick up a job that is still being processed.
	lease time.Duration
}

func New(database *db.DB, tutorSvc *tutor.Service, log *slog.Logger) *Compactor {
	return &Compactor{
		db:       database,
		tutor:    tutorSvc,
		log:      log.With("component", "compactor"),
		interval: time.Minute,
		batch:    10,
		lease:    10 * time.Minute,
	}
}

// Run processes pending lessons until ctx is cancelled.
func (c *Compactor) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		c.runOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Compactor) runOnce(ctx context.Context) {
	lessons, err := db.ClaimCompaction(ctx, c.db, c.batch, c.lease)
	if err != nil {
		if ctx.Err() == nil {
			c.log.Error("claim lessons", "err", err)
		}
		return
	}
	for _, lesson := range lessons {
		if ctx.Err() != nil {
			return
		}
		c.process(ctx, lesson)
	}
}

func (c *Compactor) process(ctx context.Context, lesson *db.Lesson) {
	log := c.log.With("lesson_id", lesson.ID)

	content, err := c.tutor.CompactLesson(ctx, lesson)
	if err == nil {
		err = c.db.WithTx(ctx, func(tx db.Querier) error {
			if err := db.CreateLessonContent(ctx, tx, lesson.ID, content); err != nil {
				return err
			}
			return db.CompleteCompaction(ctx, tx, lesson.ID)
		})
	}
	if err != nil {
		if ctx.Err() != nil {
			return // the lease expires and the job is retried
		}
		log.Warn("compaction failed", "attempt", lesson.CompactionAttempts+1, "err", err)
		metrics.CompactionJobs.WithLabelValues("error").Inc()
		if err := db.FailCompaction(ctx, c.db, lesson.ID); err != nil {
			log.Error("record failed compaction", "err", err)
		}
		return
	}
	metrics.CompactionJobs.WithLabelValues("ok").Inc()

	if err := c.tutor.TagLesson(ctx, lesson.ID, content); err != nil {
		log.Warn("tag lesson", "err", err)
	}
}
