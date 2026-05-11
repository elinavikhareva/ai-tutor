package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/elinavikhareva/ai-tutor/backend/internal/api"
	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/dbtest"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor/tutortest"
)

const password = "correct horse battery staple"

type env struct {
	t   *testing.T
	srv *httptest.Server
	db  *db.DB
	llm *tutortest.FakeLLM
}

func newEnv(t *testing.T) *env {
	t.Helper()
	database := dbtest.New(t)
	authSvc, err := auth.New([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	llm := &tutortest.FakeLLM{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := api.NewHandler(authSvc, database, tutor.New(database, llm), log)

	srv := httptest.NewServer(h.Router([]string{"http://app.test"}))
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, db: database, llm: llm}
}

// bcrypt is deliberately slow, so hash the shared test password once.
var passwordHash = sync.OnceValues(func() (string, error) { return auth.HashPassword(password) })

func (e *env) createUser(name string) {
	e.t.Helper()
	hash, err := passwordHash()
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := db.CreateUser(context.Background(), e.db, name, hash); err != nil {
		e.t.Fatal(err)
	}
}

type session struct {
	AccessToken string `json:"access_token"`
	CSRFToken   string `json:"csrf_token"`
	cookie      *http.Cookie
}

func (e *env) login(name string) *session {
	e.t.Helper()
	resp := e.do(http.MethodPost, "/api/v1/auth/login", nil, map[string]string{"username": name, "password": password})
	return e.readSession(resp)
}

func (e *env) refresh(s *session) *http.Response {
	e.t.Helper()
	req := e.request(http.MethodPost, "/api/v1/auth/refresh", nil, nil)
	req.Header.Set("X-CSRF-Token", s.CSRFToken)
	req.AddCookie(s.cookie)
	return e.send(req)
}

func (e *env) readSession(resp *http.Response) *session {
	e.t.Helper()
	if resp.StatusCode != http.StatusOK {
		e.t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var s session
	decode(e.t, resp, &s)
	for _, c := range resp.Cookies() {
		if c.Name == auth.RefreshCookieName {
			s.cookie = c
		}
	}
	if s.cookie == nil || !s.cookie.HttpOnly {
		e.t.Fatal("no httpOnly refresh cookie")
	}
	return &s
}

func (e *env) request(method, path string, s *session, body any) *http.Request {
	e.t.Helper()
	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		r = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(e.t.Context(), method, e.srv.URL+path, r)
	if err != nil {
		e.t.Fatal(err)
	}
	if s != nil {
		req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	}
	return req
}

func (e *env) send(req *http.Request) *http.Response {
	e.t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func (e *env) do(method, path string, s *session, body any) *http.Response {
	e.t.Helper()
	return e.send(e.request(method, path, s, body))
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func expectStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: status = %d, want %d (%s)", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, want, body)
	}
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)
	e.createUser("admin")

	expectStatus(t, e.do(http.MethodGet, "/api/v1/courses", nil, nil), http.StatusUnauthorized)
	expectStatus(t, e.do(http.MethodPost, "/api/v1/auth/login", nil,
		map[string]string{"username": "admin", "password": "wrong"}), http.StatusUnauthorized)
	expectStatus(t, e.do(http.MethodPost, "/api/v1/auth/login", nil,
		map[string]string{"username": "nobody", "password": password}), http.StatusUnauthorized)

	first := e.login("admin")
	expectStatus(t, e.do(http.MethodGet, "/api/v1/courses", first, nil), http.StatusOK)

	t.Run("refresh needs csrf token", func(t *testing.T) {
		req := e.request(http.MethodPost, "/api/v1/auth/refresh", nil, nil)
		req.AddCookie(first.cookie)
		expectStatus(t, e.send(req), http.StatusForbidden)
	})

	second := e.readSession(e.refresh(first))
	if second.cookie.Value == first.cookie.Value {
		t.Fatal("refresh token was not rotated")
	}

	t.Run("reusing a rotated token revokes the session", func(t *testing.T) {
		expectStatus(t, e.refresh(first), http.StatusUnauthorized)
		expectStatus(t, e.refresh(second), http.StatusUnauthorized)
	})
}

func TestLogout(t *testing.T) {
	e := newEnv(t)
	e.createUser("admin")
	s := e.login("admin")

	req := e.request(http.MethodPost, "/api/v1/auth/logout", nil, nil)
	req.Header.Set("X-CSRF-Token", s.CSRFToken)
	req.AddCookie(s.cookie)
	expectStatus(t, e.send(req), http.StatusNoContent)

	expectStatus(t, e.refresh(s), http.StatusUnauthorized)
}

type courseResponse struct {
	ID       int64 `json:"id"`
	Chapters []struct {
		Number  string `json:"number"`
		Status  string `json:"status"`
		Lessons []struct {
			ID       int64  `json:"id"`
			Position int    `json:"position"`
			Title    string `json:"title"`
		} `json:"lessons"`
	} `json:"chapters"`
}

func (e *env) createCourse(s *session) courseResponse {
	e.t.Helper()
	e.llm.ReturnJSON(`{"chapters":[
		{"title":"Goroutines","lessons":[{"title":"Overview","objective":"o"},{"title":"Scheduler","objective":"o"}]},
		{"title":"Channels","lessons":[{"title":"Unbuffered","objective":"o"}]}
	]}`)
	resp := e.do(http.MethodPost, "/api/v1/courses", s, map[string]any{"title": "Go", "motivation": "work", "goal_depth": 2})
	expectStatus(e.t, resp, http.StatusCreated)
	var created struct {
		ID int64 `json:"id"`
	}
	decode(e.t, resp, &created)

	resp = e.do(http.MethodGet, fmt.Sprintf("/api/v1/courses/%d", created.ID), s, nil)
	expectStatus(e.t, resp, http.StatusOK)
	var course courseResponse
	decode(e.t, resp, &course)
	return course
}

func TestCourseAndLessonFlow(t *testing.T) {
	e := newEnv(t)
	e.createUser("admin")
	s := e.login("admin")

	course := e.createCourse(s)
	if len(course.Chapters) != 2 || course.Chapters[1].Number != "2" || len(course.Chapters[0].Lessons) != 2 {
		t.Fatalf("unexpected course: %+v", course)
	}
	lessonID := course.Chapters[0].Lessons[0].ID
	lessonPath := fmt.Sprintf("/api/v1/lessons/%d", lessonID)

	t.Run("chat streams the tutor answer", func(t *testing.T) {
		e.llm.StreamChunks("Gorou", "tines")
		resp := e.do(http.MethodPost, lessonPath+"/chat", s, map[string]any{
			"history": []map[string]string{{"role": "user", "text": "what is a goroutine?"}},
		})
		expectStatus(t, resp, http.StatusOK)
		body, _ := io.ReadAll(resp.Body)
		for _, want := range []string{`event: chunk`, `"text":"Gorou"`, `"text":"tines"`, `event: done`} {
			if !strings.Contains(string(body), want) {
				t.Errorf("stream has no %s:\n%s", want, body)
			}
		}
	})

	t.Run("chat rejects a history ending with the tutor", func(t *testing.T) {
		resp := e.do(http.MethodPost, lessonPath+"/chat", s, map[string]any{
			"history": []map[string]string{{"role": "assistant", "text": "hi"}},
		})
		expectStatus(t, resp, http.StatusBadRequest)
	})

	t.Run("completing twice keeps one transcript", func(t *testing.T) {
		for _, answer := range []string{"first", "second"} {
			resp := e.do(http.MethodPost, lessonPath+"/complete", s, map[string]any{
				"history": []map[string]string{{"role": "user", "text": "q"}, {"role": "assistant", "text": answer}},
			})
			expectStatus(t, resp, http.StatusNoContent)
		}
		msgs, err := db.ListChatMessages(context.Background(), e.db, lessonID)
		if err != nil {
			t.Fatal(err)
		}
		if len(msgs) != 2 || msgs[1].Text != "second" {
			t.Errorf("transcript = %+v", msgs)
		}

		resp := e.do(http.MethodGet, lessonPath, s, nil)
		expectStatus(t, resp, http.StatusOK)
		var lesson struct {
			Status           string `json:"status"`
			CompactionStatus string `json:"compaction_status"`
		}
		decode(t, resp, &lesson)
		if lesson.Status != "done" || lesson.CompactionStatus != "pending" {
			t.Errorf("lesson = %+v", lesson)
		}
	})
}

func TestLessonsAreIsolatedBetweenUsers(t *testing.T) {
	e := newEnv(t)
	e.createUser("alice")
	e.createUser("bob")
	alice, bob := e.login("alice"), e.login("bob")

	course := e.createCourse(alice)
	lessonPath := fmt.Sprintf("/api/v1/lessons/%d", course.Chapters[0].Lessons[0].ID)

	expectStatus(t, e.do(http.MethodGet, fmt.Sprintf("/api/v1/courses/%d", course.ID), bob, nil), http.StatusNotFound)
	expectStatus(t, e.do(http.MethodGet, lessonPath, bob, nil), http.StatusNotFound)
	expectStatus(t, e.do(http.MethodPost, lessonPath+"/chat", bob, map[string]any{"history": []any{}}), http.StatusNotFound)
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	expectStatus(t, e.do(http.MethodGet, "/healthz", nil, nil), http.StatusOK)
	expectStatus(t, e.do(http.MethodGet, "/readyz", nil, nil), http.StatusOK)
}
