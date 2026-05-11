package db_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/dbtest"
)

type fixture struct {
	ctx     context.Context
	db      *db.DB
	userID  int64
	lessons []*db.Lesson
}

func newFixture(t *testing.T, lessons int) *fixture {
	t.Helper()
	ctx := context.Background()
	database := dbtest.New(t)

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
	f := &fixture{ctx: ctx, db: database, userID: user.ID}
	for i := range lessons {
		l, err := db.CreateLesson(ctx, database, chapter.ID, i+1, "Lesson", "objective")
		if err != nil {
			t.Fatal(err)
		}
		f.lessons = append(f.lessons, l)
	}
	return f
}

func TestGetLessonForUser(t *testing.T) {
	f := newFixture(t, 1)
	stranger, err := db.CreateUser(f.ctx, f.db, "stranger", "hash")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.GetLessonForUser(f.ctx, f.db, f.lessons[0].ID, f.userID); err != nil {
		t.Errorf("owner: %v", err)
	}
	if _, err := db.GetLessonForUser(f.ctx, f.db, f.lessons[0].ID, stranger.ID); !errors.Is(err, db.ErrNotFound) {
		t.Errorf("stranger: err = %v, want ErrNotFound", err)
	}
}

func TestReplaceChatMessages(t *testing.T) {
	f := newFixture(t, 1)
	id := f.lessons[0].ID
	first := []db.ChatMessage{{Role: "user", Text: "hi"}, {Role: "assistant", Text: "hello"}}
	second := []db.ChatMessage{{Role: "user", Text: "again"}}

	for _, msgs := range [][]db.ChatMessage{first, second} {
		if err := db.ReplaceChatMessages(f.ctx, f.db, id, msgs); err != nil {
			t.Fatal(err)
		}
	}

	got, err := db.ListChatMessages(f.ctx, f.db, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "again" || got[0].Position != 1 {
		t.Errorf("transcript = %+v, want only the second one", got)
	}
}

func TestClaimCompactionIsExclusive(t *testing.T) {
	f := newFixture(t, 20)
	for _, l := range f.lessons {
		if err := db.QueueCompaction(f.ctx, f.db, l.ID); err != nil {
			t.Fatal(err)
		}
	}

	var (
		mu      sync.Mutex
		claimed = map[int64]int{}
		wg      sync.WaitGroup
	)
	for range 4 {
		wg.Go(func() {
			lessons, err := db.ClaimCompaction(f.ctx, f.db, 10, time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, l := range lessons {
				claimed[l.ID]++
			}
		})
	}
	wg.Wait()

	if len(claimed) != len(f.lessons) {
		t.Errorf("claimed %d lessons, want %d", len(claimed), len(f.lessons))
	}
	for id, n := range claimed {
		if n > 1 {
			t.Errorf("lesson %d claimed %d times", id, n)
		}
	}
	again, err := db.ClaimCompaction(f.ctx, f.db, 10, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Errorf("leased lessons were claimed again: %d", len(again))
	}
}

func TestFailCompactionGivesUp(t *testing.T) {
	f := newFixture(t, 1)
	id := f.lessons[0].ID
	if err := db.QueueCompaction(f.ctx, f.db, id); err != nil {
		t.Fatal(err)
	}

	for range db.MaxCompactionAttempts {
		if err := db.FailCompaction(f.ctx, f.db, id); err != nil {
			t.Fatal(err)
		}
	}

	l, err := db.GetLesson(f.ctx, f.db, id)
	if err != nil {
		t.Fatal(err)
	}
	if l.CompactionStatus != "failed" || l.CompactionAttempts != db.MaxCompactionAttempts {
		t.Errorf("status %q after %d attempts", l.CompactionStatus, l.CompactionAttempts)
	}
}
