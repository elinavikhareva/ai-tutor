package tutor

import (
	"context"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

// LLM is the part of the Gemini client the tutor needs; tests use a fake.
type LLM interface {
	Generate(ctx context.Context, prompt string) (string, error)
	GenerateJSON(ctx context.Context, prompt string, schema *gemini.Schema, v any) error
	Stream(ctx context.Context, system string, history []gemini.Message) (<-chan string, <-chan error)
}

type Service struct {
	db  *db.DB
	llm LLM
}

func New(database *db.DB, llm LLM) *Service {
	return &Service{db: database, llm: llm}
}

func depthLabel(depth int) string {
	switch depth {
	case 1:
		return "surface (general understanding, no details)"
	case 3:
		return "expert (every nuance, edge cases, deep theory)"
	default:
		return "working (can apply in practice, understand why)"
	}
}
