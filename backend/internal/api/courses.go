package api

import (
	"net/http"
	"time"

	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
)

type lessonResponse struct {
	ID        int64  `json:"id"`
	Position  int    `json:"position"`
	Title     string `json:"title"`
	Objective string `json:"objective"`
	Status    string `json:"status"`
}

type chapterResponse struct {
	ID       int64            `json:"id"`
	Number   string           `json:"number"`
	Title    string           `json:"title"`
	Position int              `json:"position"`
	Status   string           `json:"status"`
	Lessons  []lessonResponse `json:"lessons"`
}

type courseResponse struct {
	ID        int64             `json:"id"`
	Title     string            `json:"title"`
	GoalDepth int               `json:"goal_depth"`
	CreatedAt time.Time         `json:"created_at"`
	Chapters  []chapterResponse `json:"chapters"`
}

func (h *Handler) suggestDirections(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title      string `json:"title"`
		Motivation string `json:"motivation"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" || req.Motivation == "" {
		writeError(w, http.StatusBadRequest, "title and motivation are required")
		return
	}

	directions, err := h.tutor.SuggestDirections(r.Context(), req.Title, req.Motivation)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"directions": directions})
}

func (h *Handler) listCourses(w http.ResponseWriter, r *http.Request) {
	courses, err := db.ListCourses(r.Context(), h.db, auth.UserID(r.Context()))
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, courses)
}

func (h *Handler) createCourse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title      string           `json:"title"`
		Motivation string           `json:"motivation"`
		GoalDepth  int              `json:"goal_depth"`
		Direction  *tutor.Direction `json:"direction"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.GoalDepth < 1 || req.GoalDepth > 3 {
		writeError(w, http.StatusBadRequest, "goal_depth must be 1, 2, or 3")
		return
	}

	plan, err := h.tutor.GeneratePlan(r.Context(), tutor.PlanInput{
		Title:      req.Title,
		Motivation: req.Motivation,
		GoalDepth:  req.GoalDepth,
		Direction:  req.Direction,
	})
	if err != nil {
		h.logger(r).Error("generate plan", "err", err)
		writeError(w, http.StatusInternalServerError, "could not generate course plan")
		return
	}

	var course *db.Course
	err = h.db.WithTx(r.Context(), func(tx db.Querier) error {
		var err error
		course, err = db.CreateCourse(r.Context(), tx, auth.UserID(r.Context()), req.Title, req.GoalDepth)
		if err != nil {
			return err
		}
		return tutor.SavePlan(r.Context(), tx, course.ID, plan)
	})
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, course)
}

func (h *Handler) getCourse(w http.ResponseWriter, r *http.Request) {
	courseID, ok := idParam(w, r)
	if !ok {
		return
	}
	course, err := db.GetCourse(r.Context(), h.db, courseID, auth.UserID(r.Context()))
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	chapters, err := db.ListChapters(r.Context(), h.db, courseID)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	lessons, err := db.ListCourseLessons(r.Context(), h.db, courseID)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	byChapter := make(map[int64][]*db.Lesson, len(chapters))
	for _, l := range lessons {
		byChapter[l.ChapterID] = append(byChapter[l.ChapterID], l)
	}

	resp := courseResponse{
		ID:        course.ID,
		Title:     course.Title,
		GoalDepth: course.GoalDepth,
		CreatedAt: course.CreatedAt,
		Chapters:  make([]chapterResponse, 0, len(chapters)),
	}
	for _, ch := range chapters {
		chLessons := byChapter[ch.ID]
		cr := chapterResponse{
			ID:       ch.ID,
			Number:   ch.Number,
			Title:    ch.Title,
			Position: ch.Position,
			Status:   chapterStatus(chLessons),
			Lessons:  make([]lessonResponse, 0, len(chLessons)),
		}
		for _, l := range chLessons {
			cr.Lessons = append(cr.Lessons, lessonResponse{
				ID:        l.ID,
				Position:  l.Position,
				Title:     l.Title,
				Objective: l.Objective,
				Status:    l.Status,
			})
		}
		resp.Chapters = append(resp.Chapters, cr)
	}
	writeJSON(w, http.StatusOK, resp)
}

func chapterStatus(lessons []*db.Lesson) string {
	if len(lessons) == 0 {
		return db.LessonPending
	}
	counts := map[string]int{}
	for _, l := range lessons {
		counts[l.Status]++
	}
	switch {
	case counts[db.LessonDone] == len(lessons):
		return db.LessonDone
	case counts[db.LessonNeedsReview] > 0:
		return db.LessonNeedsReview
	case counts[db.LessonInProgress] > 0 || counts[db.LessonDone] > 0:
		return db.LessonInProgress
	default:
		return db.LessonPending
	}
}

func (h *Handler) listKnowledge(w http.ResponseWriter, r *http.Request) {
	knowledge, err := db.ListKnowledge(r.Context(), h.db, auth.UserID(r.Context()))
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, knowledge)
}
