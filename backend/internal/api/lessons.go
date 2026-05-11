package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

const (
	maxChatMessages    = 200
	maxChatMessageSize = 8 << 10
)

type chatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

func validateHistory(history []chatMessage) error {
	if len(history) > maxChatMessages {
		return fmt.Errorf("history is limited to %d messages", maxChatMessages)
	}
	for i, m := range history {
		if m.Role != "user" && m.Role != "assistant" {
			return fmt.Errorf("message %d: unknown role %q", i+1, m.Role)
		}
		if strings.TrimSpace(m.Text) == "" || len(m.Text) > maxChatMessageSize {
			return fmt.Errorf("message %d: text must be 1 to %d bytes", i+1, maxChatMessageSize)
		}
	}
	return nil
}

func (h *Handler) getLesson(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	content, err := db.GetLessonContent(r.Context(), h.db, lesson.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		h.respondError(w, r, err)
		return
	}
	testResult, err := db.GetLatestTestResult(r.Context(), h.db, lesson.ID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		*db.Lesson
		Content    string         `json:"content,omitempty"`
		TestResult *db.TestResult `json:"last_test_result,omitempty"`
	}{lesson, content, testResult})
}

func (h *Handler) chatLesson(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	var req struct {
		History []chatMessage `json:"history"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateHistory(req.History); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if n := len(req.History); n > 0 && req.History[n-1].Role != "user" {
		writeError(w, http.StatusBadRequest, "the last message must come from the user")
		return
	}

	history := make([]gemini.Message, len(req.History))
	for i, m := range req.History {
		history[i] = gemini.Message{Role: m.Role, Text: m.Text}
	}
	textCh, errCh, err := h.tutor.StreamLesson(r.Context(), lesson, auth.UserID(r.Context()), history)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	for textCh != nil || errCh != nil {
		select {
		case text, ok := <-textCh:
			if !ok {
				textCh = nil
				continue
			}
			writeSSE(w, "chunk", map[string]string{"text": text})
			flusher.Flush()
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			h.logger(r).Error("stream lesson", "lesson_id", lesson.ID, "err", err)
			writeSSE(w, "error", map[string]string{"message": "tutor is unavailable, try again later"})
			flusher.Flush()
			return
		case <-r.Context().Done():
			return
		}
	}
	writeSSE(w, "done", map[string]bool{"done": true})
	flusher.Flush()
}

func (h *Handler) completeLesson(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	var req struct {
		History []chatMessage `json:"history"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validateHistory(req.History); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	msgs := make([]db.ChatMessage, len(req.History))
	for i, m := range req.History {
		msgs[i] = db.ChatMessage{Role: m.Role, Text: m.Text}
	}
	// Completing again replaces the transcript and recompacts the lesson.
	err := h.db.WithTx(r.Context(), func(tx db.Querier) error {
		if err := db.ReplaceChatMessages(r.Context(), tx, lesson.ID, msgs); err != nil {
			return err
		}
		if err := db.UpdateLessonStatus(r.Context(), tx, lesson.ID, db.LessonDone); err != nil {
			return err
		}
		return db.QueueCompaction(r.Context(), tx, lesson.ID)
	})
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) generateTest(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	test, err := h.tutor.GenerateTest(r.Context(), lesson.ID)
	if err != nil {
		h.respondError(w, r, err)
		return
	}

	// Expected answers stay on the server.
	type question struct {
		Question string   `json:"question"`
		Type     string   `json:"type"`
		Options  []string `json:"options,omitempty"`
	}
	questions := make([]question, 0, len(test.Questions))
	for _, q := range test.Questions {
		questions = append(questions, question{Question: q.Question, Type: q.Type, Options: q.Options})
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": questions})
}

func (h *Handler) submitTest(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	var req struct {
		Answers []string `json:"answers"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.tutor.EvaluateAnswers(r.Context(), lesson.ID, auth.UserID(r.Context()), req.Answers)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) updateLessonStatus(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !db.ValidLessonStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "invalid status value")
		return
	}
	if err := db.UpdateLessonStatus(r.Context(), h.db, lesson.ID, req.Status); err != nil {
		h.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateLessonNotes(w http.ResponseWriter, r *http.Request) {
	lesson, ok := h.lessonParam(w, r)
	if !ok {
		return
	}
	var req struct {
		Notes string `json:"notes"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := db.UpdateLessonNotes(r.Context(), h.db, lesson.ID, req.Notes); err != nil {
		h.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
