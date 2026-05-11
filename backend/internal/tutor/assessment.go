package tutor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

const (
	maxTags = 10
	// A concept the student got wrong gets this much less confidence than
	// the overall score.
	weakTagPenalty = 30

	QuestionMultipleChoice = "multiple_choice"
	QuestionOpen           = "open"
)

type Test struct {
	Questions []Question `json:"questions"`
}

type Question struct {
	Question string   `json:"question"`
	Type     string   `json:"type"`
	Options  []string `json:"options,omitempty"`
	Answer   string   `json:"answer"`
}

type Evaluation struct {
	Score    int      `json:"score"`
	Feedback string   `json:"feedback"`
	WeakTags []string `json:"weak_tags"`
}

var tagsSchema = gemini.Object(map[string]*gemini.Schema{
	"tags": gemini.Array(gemini.String()),
})

var testSchema = gemini.Object(map[string]*gemini.Schema{
	"questions": gemini.Array(gemini.Object(map[string]*gemini.Schema{
		"question": gemini.String(),
		"type":     gemini.Enum(QuestionMultipleChoice, QuestionOpen),
		"options":  gemini.Array(gemini.String()),
		"answer":   gemini.String(),
	}, "options")),
})

var evaluationSchema = gemini.Object(map[string]*gemini.Schema{
	"score":     gemini.Integer(),
	"feedback":  gemini.String(),
	"weak_tags": gemini.Array(gemini.String()),
})

func (s *Service) TagLesson(ctx context.Context, lessonID int64, content string) error {
	prompt := fmt.Sprintf(`Extract the key concepts of this lesson as short tags (2-4 words each).

Content:
%s`, content)

	var result struct {
		Tags []string `json:"tags"`
	}
	if err := s.llm.GenerateJSON(ctx, prompt, tagsSchema, &result); err != nil {
		return fmt.Errorf("extract tags: %w", err)
	}
	return db.AddLessonTags(ctx, s.db, lessonID, normalizeTags(result.Tags))
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	var out []string
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
		if len(out) == maxTags {
			break
		}
	}
	return out
}

func (s *Service) GenerateTest(ctx context.Context, lessonID int64) (*Test, error) {
	lesson, err := db.GetLesson(ctx, s.db, lessonID)
	if err != nil {
		return nil, fmt.Errorf("get lesson: %w", err)
	}
	tags, err := db.ListLessonTags(ctx, s.db, lessonID)
	if err != nil {
		return nil, fmt.Errorf("list lesson tags: %w", err)
	}
	content, err := db.GetLessonContent(ctx, s.db, lessonID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, fmt.Errorf("get lesson content: %w", err)
	}

	prompt := fmt.Sprintf(`Create a test for the lesson %q.
Objective: %s
Key concepts: %s
Lesson content: %s

Write 3-5 questions mixing multiple choice and open questions.
For multiple choice, "answer" must be exactly one of the options.
For open questions, "answer" lists the expected key points.`,
		lesson.Title,
		lesson.Objective,
		strings.Join(tags, ", "),
		content,
	)

	var test Test
	if err := s.llm.GenerateJSON(ctx, prompt, testSchema, &test); err != nil {
		return nil, fmt.Errorf("generate test: %w", err)
	}
	test.Questions = slices.DeleteFunc(test.Questions, func(q Question) bool { return !q.valid() })
	if len(test.Questions) == 0 {
		return nil, errors.New("generate test: no valid questions")
	}

	data, err := json.Marshal(test.Questions)
	if err != nil {
		return nil, fmt.Errorf("marshal test: %w", err)
	}
	if err := db.CreateLessonTest(ctx, s.db, lessonID, string(data)); err != nil {
		return nil, fmt.Errorf("save test: %w", err)
	}
	return &test, nil
}

func (q Question) valid() bool {
	if strings.TrimSpace(q.Question) == "" || strings.TrimSpace(q.Answer) == "" {
		return false
	}
	switch q.Type {
	case QuestionOpen:
		return true
	case QuestionMultipleChoice:
		return len(q.Options) >= 2 && slices.Contains(q.Options, q.Answer)
	default:
		return false
	}
}

func (s *Service) EvaluateAnswers(ctx context.Context, lessonID, userID int64, answers []string) (*Evaluation, error) {
	stored, err := db.GetLatestLessonTest(ctx, s.db, lessonID)
	if err != nil {
		return nil, fmt.Errorf("load test: %w", err)
	}
	var questions []Question
	if err := json.Unmarshal([]byte(stored), &questions); err != nil {
		return nil, fmt.Errorf("parse stored test: %w", err)
	}

	var qa strings.Builder
	for i, q := range questions {
		var answer string
		if i < len(answers) {
			answer = answers[i]
		}
		fmt.Fprintf(&qa, "Q%d: %s\nExpected: %s\nStudent answered: %s\n\n", i+1, q.Question, q.Answer, answer)
	}

	tags, err := db.ListLessonTags(ctx, s.db, lessonID)
	if err != nil {
		return nil, fmt.Errorf("list lesson tags: %w", err)
	}

	prompt := fmt.Sprintf(`Evaluate the student's answers for a lesson on: %s

%s
Give a score from 0 to 100, brief encouraging feedback, and the concepts the
student struggled with (use the lesson's concepts where they apply).`,
		strings.Join(tags, ", "),
		qa.String(),
	)

	var eval Evaluation
	if err := s.llm.GenerateJSON(ctx, prompt, evaluationSchema, &eval); err != nil {
		return nil, fmt.Errorf("evaluate answers: %w", err)
	}
	eval.Score = min(max(eval.Score, 0), 100)

	err = s.db.WithTx(ctx, func(tx db.Querier) error {
		for _, tag := range tags {
			if err := db.UpsertKnowledge(ctx, tx, userID, tag, tagConfidence(tag, eval)); err != nil {
				return err
			}
		}
		return db.CreateTestResult(ctx, tx, lessonID, eval.Score, eval.Feedback)
	})
	if err != nil {
		return nil, fmt.Errorf("save test result: %w", err)
	}
	return &eval, nil
}

func tagConfidence(tag string, eval Evaluation) int {
	for _, weak := range eval.WeakTags {
		if strings.EqualFold(tag, strings.TrimSpace(weak)) {
			return max(0, eval.Score-weakTagPenalty)
		}
	}
	return eval.Score
}
