package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
)

type Handler struct {
	auth  *auth.Service
	db    *db.DB
	tutor *tutor.Service
	log   *slog.Logger
}

func NewHandler(authSvc *auth.Service, database *db.DB, tutorSvc *tutor.Service, log *slog.Logger) *Handler {
	return &Handler{auth: authSvc, db: database, tutor: tutorSvc, log: log}
}

func (h *Handler) Router(allowedOrigins []string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)

	r.Get("/healthz", h.healthz)
	r.Get("/readyz", h.readyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(h.logRequests, middleware.Recoverer, bodyLimit(1<<20), cors(allowedOrigins))

		r.Route("/auth", func(r chi.Router) {
			r.With(loginRateLimit(5, time.Minute)).Post("/login", h.login)
			r.Post("/refresh", h.refresh)
			r.Post("/logout", h.logout)
		})
		r.Group(func(r chi.Router) {
			r.Use(h.auth.Middleware)
			r.Get("/courses", h.listCourses)
			r.Post("/courses/directions", h.suggestDirections)
			r.Post("/courses", h.createCourse)
			r.Get("/courses/{id}", h.getCourse)
			r.Get("/lessons/{id}", h.getLesson)
			r.Post("/lessons/{id}/chat", h.chatLesson)
			r.Post("/lessons/{id}/complete", h.completeLesson)
			r.Post("/lessons/{id}/test", h.generateTest)
			r.Post("/lessons/{id}/submit", h.submitTest)
			r.Patch("/lessons/{id}/status", h.updateLessonStatus)
			r.Patch("/lessons/{id}/notes", h.updateLessonNotes)
			r.Get("/knowledge", h.listKnowledge)
		})
	})
	return r
}
