package tutor

import (
	"context"
	"fmt"
	"strings"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

const (
	weakTagThreshold = 60
	maxHistoryBytes  = 32 << 10
)

func (s *Service) StreamLesson(ctx context.Context, lesson *db.Lesson, userID int64, history []gemini.Message) (<-chan string, <-chan error, error) {
	system, err := s.systemPrompt(ctx, lesson, userID)
	if err != nil {
		return nil, nil, err
	}
	history = trimHistory(history, maxHistoryBytes)
	if len(history) == 0 {
		history = []gemini.Message{{Role: "user", Text: "Please teach me this lesson."}}
	}
	text, errs := s.llm.Stream(ctx, system, history)
	return text, errs, nil
}

// trimHistory keeps the latest messages that fit into budget bytes, always
// keeping the last one. The result starts with a user turn, as Gemini expects.
func trimHistory(history []gemini.Message, budget int) []gemini.Message {
	start := len(history)
	for start > 0 {
		size := len(history[start-1].Text)
		if size > budget && start < len(history) {
			break
		}
		budget -= size
		start--
	}
	for start < len(history) && history[start].Role != "user" {
		start++
	}
	return history[start:]
}

func (s *Service) systemPrompt(ctx context.Context, lesson *db.Lesson, userID int64) (string, error) {
	depth, err := db.GetLessonGoalDepth(ctx, s.db, lesson.ID)
	if err != nil {
		return "", fmt.Errorf("get goal depth: %w", err)
	}
	weakTags, err := db.ListWeakTags(ctx, s.db, userID, weakTagThreshold)
	if err != nil {
		return "", fmt.Errorf("list weak tags: %w", err)
	}

	return fmt.Sprintf(`You are a personal tutor. Teach clearly and engagingly.

LESSON: %s
OBJECTIVE: %s
DEPTH: %s
WEAK CONCEPTS (reinforce these): %s

Rules:
- Teach exactly to the objective, no more and no less
- Match the depth level
- Use examples and analogies where helpful
- End every response by asking: (1) test me, (2) go deeper, (3) next lesson
- No padding`,
		lesson.Title,
		lesson.Objective,
		depthLabel(depth),
		strings.Join(weakTags, ", "),
	), nil
}
