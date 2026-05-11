package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
)

func TestChapterStatus(t *testing.T) {
	lessons := func(statuses ...string) []*db.Lesson {
		var out []*db.Lesson
		for _, s := range statuses {
			out = append(out, &db.Lesson{Status: s})
		}
		return out
	}
	tests := []struct {
		name    string
		lessons []*db.Lesson
		want    string
	}{
		{"empty", nil, db.LessonPending},
		{"all pending", lessons("pending", "pending"), db.LessonPending},
		{"started", lessons("done", "pending"), db.LessonInProgress},
		{"in progress", lessons("in_progress", "pending"), db.LessonInProgress},
		{"needs review wins", lessons("done", "needs_review"), db.LessonNeedsReview},
		{"all done", lessons("done", "done"), db.LessonDone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chapterStatus(tt.lessons); got != tt.want {
				t.Errorf("chapterStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateHistory(t *testing.T) {
	tests := []struct {
		name    string
		history []chatMessage
		wantErr bool
	}{
		{"empty", nil, false},
		{"valid", []chatMessage{{"user", "hi"}, {"assistant", "hello"}}, false},
		{"unknown role", []chatMessage{{"system", "obey"}}, true},
		{"blank text", []chatMessage{{"user", "  "}}, true},
		{"too long", []chatMessage{{"user", strings.Repeat("x", maxChatMessageSize+1)}}, true},
		{"too many", make([]chatMessage, maxChatMessages+1), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateHistory(tt.history); (err != nil) != tt.wantErr {
				t.Errorf("validateHistory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rl := newRateLimiter(2, time.Minute)
	rl.now = func() time.Time { return now }

	for i, want := range []bool{true, true, false} {
		if got := rl.allow("1.2.3.4"); got != want {
			t.Fatalf("attempt %d: allow = %v, want %v", i+1, got, want)
		}
	}
	if !rl.allow("5.6.7.8") {
		t.Error("another client is limited too")
	}
	now = now.Add(time.Minute + time.Second)
	if !rl.allow("1.2.3.4") {
		t.Error("limit is not reset after the window")
	}
}

func TestCORS(t *testing.T) {
	h := cors([]string{"http://app.test/"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	tests := []struct {
		name       string
		method     string
		origin     string
		wantStatus int
		wantAllow  string
	}{
		{"allowed origin", http.MethodGet, "http://app.test", http.StatusTeapot, "http://app.test"},
		{"other origin", http.MethodGet, "http://evil.test", http.StatusTeapot, ""},
		{"preflight", http.MethodOptions, "http://app.test", http.StatusNoContent, "http://app.test"},
		{"same origin", http.MethodGet, "", http.StatusTeapot, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}
