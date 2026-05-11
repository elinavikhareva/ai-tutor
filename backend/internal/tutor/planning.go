package tutor

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

const (
	maxDirections        = 4
	maxChapters          = 12
	maxLessonsPerChapter = 12
)

type Direction struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type CoursePlan struct {
	Chapters []ChapterPlan `json:"chapters"`
}

type ChapterPlan struct {
	Title   string       `json:"title"`
	Lessons []LessonPlan `json:"lessons"`
}

type LessonPlan struct {
	Title     string `json:"title"`
	Objective string `json:"objective"`
}

type PlanInput struct {
	Title      string
	Motivation string
	GoalDepth  int
	Direction  *Direction
}

var directionsSchema = gemini.Object(map[string]*gemini.Schema{
	"directions": gemini.Array(gemini.Object(map[string]*gemini.Schema{
		"title":       gemini.String(),
		"description": gemini.String(),
	})),
})

var planSchema = gemini.Object(map[string]*gemini.Schema{
	"chapters": gemini.Array(gemini.Object(map[string]*gemini.Schema{
		"title": gemini.String(),
		"lessons": gemini.Array(gemini.Object(map[string]*gemini.Schema{
			"title":     gemini.String(),
			"objective": gemini.String(),
		})),
	})),
})

func (s *Service) SuggestDirections(ctx context.Context, title, motivation string) ([]Direction, error) {
	prompt := fmt.Sprintf(`You are a curriculum designer. A student wants to learn about %q.
Their goal: %s

Suggest 1-4 distinct learning directions.
- If the topic is narrow and the goal is clear, suggest exactly 1 direction
- If there are meaningfully different angles, suggest 2-4 directions
- Each direction must be concrete and distinct, not variations of the same thing
- The description says in 1-2 sentences what the student will be able to do after`, title, motivation)

	var result struct {
		Directions []Direction `json:"directions"`
	}
	if err := s.llm.GenerateJSON(ctx, prompt, directionsSchema, &result); err != nil {
		return nil, fmt.Errorf("suggest directions: %w", err)
	}
	directions := slices.DeleteFunc(result.Directions, func(d Direction) bool {
		return strings.TrimSpace(d.Title) == ""
	})
	if len(directions) == 0 {
		return nil, errors.New("suggest directions: empty result")
	}
	return directions[:min(len(directions), maxDirections)], nil
}

func (s *Service) GeneratePlan(ctx context.Context, in PlanInput) (*CoursePlan, error) {
	lessonsPerChapter := map[int]string{
		1: "3-4 overview lessons",
		2: "4-6 lessons",
		3: "6-10 detailed lessons",
	}[in.GoalDepth]

	var b strings.Builder
	fmt.Fprintf(&b, "You are a curriculum designer. Create a course plan for: %q\n\n", in.Title)
	if in.Motivation != "" {
		fmt.Fprintf(&b, "Student's goal: %s\n", in.Motivation)
	}
	if in.Direction != nil && in.Direction.Title != "" {
		fmt.Fprintf(&b, "Learning direction: %s\nDirection context: %s\n", in.Direction.Title, in.Direction.Description)
	}
	fmt.Fprintf(&b, "Depth level: %s\nLessons per chapter: %s\n", depthLabel(in.GoalDepth), lessonsPerChapter)
	b.WriteString(`
Rules:
- 4-7 chapters, listed in teaching order
- Stay focused on the student's goal and the chosen direction, if any
- The FIRST lesson of EVERY chapter is an overview: broad picture, key concepts, why this matters
- Remaining lessons dive into specific subtopics in logical order
- "objective" is 1-3 concrete verifiable theses, NOT a paraphrase of the title
- Titles are specific and actionable`)

	var plan CoursePlan
	if err := s.llm.GenerateJSON(ctx, b.String(), planSchema, &plan); err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}
	if err := plan.validate(); err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}
	return &plan, nil
}

func (p *CoursePlan) validate() error {
	if len(p.Chapters) == 0 || len(p.Chapters) > maxChapters {
		return fmt.Errorf("invalid plan: %d chapters", len(p.Chapters))
	}
	for i, ch := range p.Chapters {
		if strings.TrimSpace(ch.Title) == "" {
			return fmt.Errorf("invalid plan: chapter %d has no title", i+1)
		}
		if len(ch.Lessons) == 0 || len(ch.Lessons) > maxLessonsPerChapter {
			return fmt.Errorf("invalid plan: chapter %d has %d lessons", i+1, len(ch.Lessons))
		}
		for j, l := range ch.Lessons {
			if strings.TrimSpace(l.Title) == "" {
				return fmt.Errorf("invalid plan: lesson %d.%d has no title", i+1, j+1)
			}
		}
	}
	return nil
}

// SavePlan numbers chapters and lessons itself rather than trusting the model.
func SavePlan(ctx context.Context, q db.Querier, courseID int64, plan *CoursePlan) error {
	for i, ch := range plan.Chapters {
		chapter, err := db.CreateChapter(ctx, q, courseID, strconv.Itoa(i+1), ch.Title, i+1)
		if err != nil {
			return fmt.Errorf("create chapter %d: %w", i+1, err)
		}
		for j, l := range ch.Lessons {
			if _, err := db.CreateLesson(ctx, q, chapter.ID, j+1, l.Title, l.Objective); err != nil {
				return fmt.Errorf("create lesson %d.%d: %w", i+1, j+1, err)
			}
		}
	}
	return nil
}
