package tutor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/dbtest"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor/tutortest"
)

const planJSON = `{"chapters":[
	{"title":"Goroutines","lessons":[{"title":"Overview","objective":"o1"},{"title":"Scheduler","objective":"o2"}]},
	{"title":"Channels","lessons":[{"title":"Unbuffered","objective":"o3"}]}
]}`

func setup(t *testing.T) (context.Context, *db.DB, *tutortest.FakeLLM, *tutor.Service, int64) {
	t.Helper()
	ctx := context.Background()
	database := dbtest.New(t)
	llm := &tutortest.FakeLLM{}
	user, err := db.CreateUser(ctx, database, "student", "hash")
	if err != nil {
		t.Fatal(err)
	}
	return ctx, database, llm, tutor.New(database, llm), user.ID
}

func createCourse(t *testing.T, ctx context.Context, database *db.DB, svc *tutor.Service, userID int64) int64 {
	t.Helper()
	plan, err := svc.GeneratePlan(ctx, tutor.PlanInput{Title: "Go", Motivation: "work", GoalDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	var courseID int64
	err = database.WithTx(ctx, func(tx db.Querier) error {
		course, err := db.CreateCourse(ctx, tx, userID, "Go", 2)
		if err != nil {
			return err
		}
		courseID = course.ID
		return tutor.SavePlan(ctx, tx, course.ID, plan)
	})
	if err != nil {
		t.Fatal(err)
	}
	return courseID
}

func TestGenerateAndSavePlan(t *testing.T) {
	ctx, database, llm, svc, userID := setup(t)
	llm.ReturnJSON(planJSON)

	courseID := createCourse(t, ctx, database, svc, userID)

	if prompt := llm.Prompts()[0]; !strings.Contains(prompt, "Student's goal: work") {
		t.Errorf("prompt does not mention the goal:\n%s", prompt)
	}
	chapters, err := db.ListChapters(ctx, database, courseID)
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 || chapters[1].Number != "2" || chapters[1].Title != "Channels" {
		t.Fatalf("unexpected chapters: %+v", chapters)
	}
	lessons, err := db.ListCourseLessons(ctx, database, courseID)
	if err != nil {
		t.Fatal(err)
	}
	var positions []int
	for _, l := range lessons {
		positions = append(positions, l.Position)
	}
	if got := len(lessons); got != 3 || positions[0] != 1 || positions[1] != 2 || positions[2] != 1 {
		t.Errorf("lessons numbered %v, want [1 2 1]", positions)
	}
}

func TestGeneratePlanRejectsInvalidPlan(t *testing.T) {
	ctx, _, llm, svc, _ := setup(t)
	llm.ReturnJSON(`{"chapters":[]}`)

	if _, err := svc.GeneratePlan(ctx, tutor.PlanInput{Title: "Go", GoalDepth: 2}); err == nil {
		t.Fatal("expected an error for an empty plan")
	}
}

func TestTestAndEvaluation(t *testing.T) {
	ctx, database, llm, svc, userID := setup(t)
	llm.ReturnJSON(planJSON)
	courseID := createCourse(t, ctx, database, svc, userID)
	lessons, err := db.ListCourseLessons(ctx, database, courseID)
	if err != nil {
		t.Fatal(err)
	}
	lessonID := lessons[0].ID

	llm.ReturnJSON(`{"tags":["Goroutines","channels","goroutines"]}`)
	if err := svc.TagLesson(ctx, lessonID, "lecture"); err != nil {
		t.Fatal(err)
	}

	llm.ReturnJSON(`{"questions":[
		{"question":"Cost of a goroutine?","type":"open","answer":"a few KB"},
		{"question":"Broken","type":"multiple_choice","options":["a"],"answer":"a"}
	]}`)
	test, err := svc.GenerateTest(ctx, lessonID)
	if err != nil {
		t.Fatal(err)
	}
	if len(test.Questions) != 1 {
		t.Fatalf("got %d questions, want the invalid one dropped", len(test.Questions))
	}

	llm.ReturnJSON(`{"score":130,"feedback":"good","weak_tags":["channels"]}`)
	eval, err := svc.EvaluateAnswers(ctx, lessonID, userID, []string{"2KB"})
	if err != nil {
		t.Fatal(err)
	}
	if eval.Score != 100 {
		t.Errorf("score = %d, want it clamped to 100", eval.Score)
	}

	knowledge, err := db.ListKnowledge(ctx, database, userID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for _, k := range knowledge {
		got[k.Tag] = k.Confidence
	}
	if got["goroutines"] != 100 || got["channels"] != 70 || len(got) != 2 {
		t.Errorf("knowledge = %v", got)
	}
}

func TestStreamLessonReportsPromptErrors(t *testing.T) {
	ctx, _, _, svc, userID := setup(t)
	missing := &db.Lesson{ID: 999}

	_, _, err := svc.StreamLesson(ctx, missing, userID, nil)
	if !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
