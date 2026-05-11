package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeSSE(w http.ResponseWriter, event string, v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
}

func (h *Handler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	h.logger(r).Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func (h *Handler) logger(r *http.Request) *slog.Logger {
	return h.log.With("request_id", middleware.GetReqID(r.Context()))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func (h *Handler) lessonParam(w http.ResponseWriter, r *http.Request) (*db.Lesson, bool) {
	id, ok := idParam(w, r)
	if !ok {
		return nil, false
	}
	lesson, err := db.GetLessonForUser(r.Context(), h.db, id, auth.UserID(r.Context()))
	if err != nil {
		h.respondError(w, r, err)
		return nil, false
	}
	return lesson, true
}
