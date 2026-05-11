package tutor

import (
	"context"
	"fmt"
	"strings"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
)

func (s *Service) CompactLesson(ctx context.Context, lesson *db.Lesson) (string, error) {
	msgs, err := db.ListChatMessages(ctx, s.db, lesson.ID)
	if err != nil {
		return "", fmt.Errorf("list chat messages: %w", err)
	}

	var chat strings.Builder
	for _, m := range msgs {
		fmt.Fprintf(&chat, "[%s]: %s\n\n", m.Role, m.Text)
	}

	prompt := fmt.Sprintf(`You are converting a tutoring chat into a clean textbook section.

Lesson: %s
Objective: %s

Chat history:
%s

Rules:
- Keep the FINAL, most complete version of each explanation (if student asked to clarify multiple times, keep the last answer)
- Remove all meta-dialogue (greetings, "shall we test?", student confirmations)
- Output clean Markdown suitable for a textbook section
- Use headers, bullet points, code blocks where appropriate
- No padding, no conversational filler`,
		lesson.Title,
		lesson.Objective,
		chat.String(),
	)

	content, err := s.llm.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("compact lesson: %w", err)
	}
	return content, nil
}
