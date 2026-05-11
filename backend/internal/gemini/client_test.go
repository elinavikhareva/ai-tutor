package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := NewClient("test-key")
	c.baseURL = srv.URL
	c.backoff = []time.Duration{time.Millisecond, time.Millisecond}
	return c
}

func reply(w http.ResponseWriter, text string) {
	fmt.Fprintf(w, `{"candidates":[{"content":{"parts":[{"text":%q}]}}]}`, text)
}

func TestGenerateJSON(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-goog-api-key"); got != "test-key" {
			t.Errorf("api key header = %q", got)
		}
		if r.URL.Query().Has("key") {
			t.Error("api key leaked into the query string")
		}
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		cfg := req.GenerationConfig
		if cfg.ResponseMIMEType != "application/json" || cfg.ResponseSchema == nil {
			t.Errorf("structured output not requested: %+v", cfg)
		}
		reply(w, `{"tags":["goroutines","channels"]}`)
	})

	var got struct{ Tags []string }
	schema := Object(map[string]*Schema{"tags": Array(String())})
	if err := c.GenerateJSON(context.Background(), "prompt", schema, &got); err != nil {
		t.Fatal(err)
	}
	if want := []string{"goroutines", "channels"}; !slices.Equal(got.Tags, want) {
		t.Errorf("tags = %q, want %q", got.Tags, want)
	}
}

func TestGenerateRetries(t *testing.T) {
	tests := []struct {
		name      string
		statuses  []int
		wantCalls int32
		wantErr   bool
	}{
		{"success", []int{200}, 1, false},
		{"recovers from 503", []int{503, 200}, 2, false},
		{"recovers from 429", []int{429, 429, 200}, 3, false},
		{"gives up", []int{500, 500, 500}, 3, true},
		{"no retry on 400", []int{400}, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				status := tt.statuses[calls.Add(1)-1]
				if status != http.StatusOK {
					w.WriteHeader(status)
					fmt.Fprint(w, `{"error":{"message":"nope"}}`)
					return
				}
				reply(w, "hello")
			})

			text, err := c.Generate(context.Background(), "prompt")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && text != "hello" {
				t.Errorf("text = %q", text)
			}
			if got := calls.Load(); got != tt.wantCalls {
				t.Errorf("calls = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestGenerateStopsOnCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c.backoff = []time.Duration{time.Hour}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.Generate(ctx, "prompt"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want deadline exceeded", err)
	}
}

func TestStream(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ":streamGenerateContent") || r.URL.Query().Get("alt") != "sse" {
			t.Errorf("unexpected url %s", r.URL)
		}
		for _, chunk := range []string{"Hel", "lo"} {
			fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":%q}]}}]}\n\n", chunk)
		}
		fmt.Fprint(w, "data: {\"error\":{\"message\":\"quota\"}}\n\n")
	})

	textCh, errCh := c.Stream(context.Background(), "system", []Message{{Role: "user", Text: "hi"}})
	var got strings.Builder
	for text := range textCh {
		got.WriteString(text)
	}
	if got.String() != "Hello" {
		t.Errorf("streamed %q, want %q", got.String(), "Hello")
	}
	if err := <-errCh; err == nil || !strings.Contains(err.Error(), "quota") {
		t.Errorf("err = %v, want the error chunk", err)
	}
}

func TestBuildRequestMapsRoles(t *testing.T) {
	req := buildRequest("be nice", []Message{{Role: "user", Text: "q"}, {Role: "assistant", Text: "a"}})
	if req.Contents[0].Role != "user" || req.Contents[1].Role != "model" {
		t.Errorf("roles = %q, %q", req.Contents[0].Role, req.Contents[1].Role)
	}
	if req.SystemInstruction == nil || req.SystemInstruction.Parts[0].Text != "be nice" {
		t.Error("system instruction is missing")
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := map[string]time.Duration{"": 0, "3": 3 * time.Second, "-1": 0, "soon": 0}
	for in, want := range tests {
		if got := parseRetryAfter(in); got != want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestObjectSchemaRequired(t *testing.T) {
	s := Object(map[string]*Schema{"b": String(), "a": String(), "opt": String()}, "opt")
	if want := []string{"a", "b"}; !slices.Equal(s.Required, want) {
		t.Errorf("required = %q, want %q", s.Required, want)
	}
}
