package compactor

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/dbtest"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor/tutortest"
)

func setup(t *testing.T) (context.Context, *db.DB, *tutortest.FakeLLM, *Compactor, int64) {
	t.Helper()
	ctx := context.Background()
	database := dbtest.New(t)
	llm := &tutortest.FakeLLM{}
	c := New(database, tutor.New(database, llm), slog.New(slog.NewTextHandler(io.Discard, nil)))

	user, err := db.CreateUser(ctx, database, "student", "hash")
	if err != nil {
		t.Fatal(err)
	}
	course, err := db.CreateCourse(ctx, database, user.ID, "Go", 2)
	if err != nil {
		t.Fatal(err)
	}
	chapter, err := db.CreateChapter(ctx, database, course.ID, "1", "Basics", 1)
	if err != nil {
		t.Fatal(err)
	}
	lesson, err := db.CreateLesson(ctx, database, chapter.ID, 1, "Goroutines", "objective")
	if err != nil {
		t.Fatal(err)
	}
	err = db.ReplaceChatMessages(ctx, database, lesson.ID, []db.ChatMessage{
		{Role: "user", Text: "what is a goroutine?"},
		{Role: "assistant", Text: "a lightweight thread"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.QueueCompaction(ctx, database, lesson.ID); err != nil {
		t.Fatal(err)
	}
	return ctx, database, llm, c, lesson.ID
}

func TestCompactsLesson(t *testing.T) {
	ctx, database, llm, c, lessonID := setup(t)
	llm.ReturnText("# Goroutines\nA lightweight thread.")
	llm.ReturnJSON(`{"tags":["goroutines"]}`)

	c.runOnce(ctx)

	content, err := db.GetLessonContent(ctx, database, lessonID)
	if err != nil {
		t.Fatal(err)
	}
	if content != "# Goroutines\nA lightweight thread." {
		t.Errorf("content = %q", content)
	}
	lesson, err := db.GetLesson(ctx, database, lessonID)
	if err != nil {
		t.Fatal(err)
	}
	if lesson.CompactionStatus != "done" {
		t.Errorf("status = %q, want done", lesson.CompactionStatus)
	}
	tags, err := db.ListLessonTags(ctx, database, lessonID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != "goroutines" {
		t.Errorf("tags = %q", tags)
	}
}

func TestFailedCompactionIsRetriedLater(t *testing.T) {
	ctx, database, llm, c, lessonID := setup(t)
	llm.Fail(errors.New("model is down"))

	c.runOnce(ctx)

	lesson, err := db.GetLesson(ctx, database, lessonID)
	if err != nil {
		t.Fatal(err)
	}
	if lesson.CompactionStatus != "pending" || lesson.CompactionAttempts != 1 {
		t.Errorf("status %q, attempts %d", lesson.CompactionStatus, lesson.CompactionAttempts)
	}
	// The retry is delayed, so an immediate second pass must not pick it up.
	claimed, err := db.ClaimCompaction(ctx, database, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Error("failed lesson was retried without backoff")
	}
}
