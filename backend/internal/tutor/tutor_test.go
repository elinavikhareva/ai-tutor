package tutor

import (
	"slices"
	"strings"
	"testing"

	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

func msg(role, text string) gemini.Message { return gemini.Message{Role: role, Text: text} }

func TestTrimHistory(t *testing.T) {
	u1, a1, u2 := msg("user", "aaaa"), msg("assistant", "bbbb"), msg("user", "cccc")
	long := msg("user", strings.Repeat("x", 100))

	tests := []struct {
		name    string
		history []gemini.Message
		budget  int
		want    []gemini.Message
	}{
		{"fits", []gemini.Message{u1, a1, u2}, 100, []gemini.Message{u1, a1, u2}},
		{"drops oldest", []gemini.Message{u1, a1, u2}, 8, []gemini.Message{u2}},
		{"starts with user", []gemini.Message{u1, a1, u2}, 9, []gemini.Message{u2}},
		{"keeps oversized last message", []gemini.Message{u1, long}, 10, []gemini.Message{long}},
		{"empty", nil, 10, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimHistory(tt.history, tt.budget)
			if !slices.Equal(got, tt.want) {
				t.Errorf("trimHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCoursePlanValidate(t *testing.T) {
	lesson := LessonPlan{Title: "Overview", Objective: "o"}
	chapter := ChapterPlan{Title: "Basics", Lessons: []LessonPlan{lesson}}

	tests := []struct {
		name    string
		plan    CoursePlan
		wantErr bool
	}{
		{"valid", CoursePlan{Chapters: []ChapterPlan{chapter}}, false},
		{"no chapters", CoursePlan{}, true},
		{"too many chapters", CoursePlan{Chapters: slices.Repeat([]ChapterPlan{chapter}, maxChapters+1)}, true},
		{"untitled chapter", CoursePlan{Chapters: []ChapterPlan{{Title: " ", Lessons: []LessonPlan{lesson}}}}, true},
		{"empty chapter", CoursePlan{Chapters: []ChapterPlan{{Title: "Basics"}}}, true},
		{"untitled lesson", CoursePlan{Chapters: []ChapterPlan{{Title: "Basics", Lessons: []LessonPlan{{}}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.plan.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQuestionValid(t *testing.T) {
	tests := []struct {
		name string
		q    Question
		want bool
	}{
		{"open", Question{Question: "Why?", Type: QuestionOpen, Answer: "because"}, true},
		{"choice", Question{Question: "Which?", Type: QuestionMultipleChoice, Options: []string{"a", "b"}, Answer: "b"}, true},
		{"answer not an option", Question{Question: "Which?", Type: QuestionMultipleChoice, Options: []string{"a", "b"}, Answer: "B"}, false},
		{"single option", Question{Question: "Which?", Type: QuestionMultipleChoice, Options: []string{"a"}, Answer: "a"}, false},
		{"unknown type", Question{Question: "?", Type: "essay", Answer: "x"}, false},
		{"no answer", Question{Question: "Why?", Type: QuestionOpen}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.q.valid(); got != tt.want {
				t.Errorf("valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	got := normalizeTags([]string{" Goroutines ", "goroutines", "", "Channels"})
	if want := []string{"goroutines", "channels"}; !slices.Equal(got, want) {
		t.Errorf("normalizeTags() = %q, want %q", got, want)
	}
	many := strings.Split("a b c d e f g h i j k l", " ")
	if got := normalizeTags(many); len(got) != maxTags {
		t.Errorf("kept %d tags, want %d", len(got), maxTags)
	}
}

func TestTagConfidence(t *testing.T) {
	eval := Evaluation{Score: 80, WeakTags: []string{" Channels "}}
	if got := tagConfidence("channels", eval); got != 50 {
		t.Errorf("weak tag confidence = %d, want 50", got)
	}
	if got := tagConfidence("goroutines", eval); got != 80 {
		t.Errorf("confidence = %d, want 80", got)
	}
	if got := tagConfidence("channels", Evaluation{Score: 10, WeakTags: []string{"channels"}}); got != 0 {
		t.Errorf("confidence = %d, want it clamped to 0", got)
	}
}
