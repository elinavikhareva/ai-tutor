package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/elinavikhareva/ai-tutor/backend/internal/metrics"
)

const (
	defaultBaseURL  = "https://generativelanguage.googleapis.com/v1beta/models"
	model           = "gemini-2.5-flash-lite"
	defaultTimeout  = 3 * time.Minute
	temperature     = 0.7
	maxOutputTokens = 8192
)

type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
	// backoff holds the base delay before each retry; its length is the
	// number of retries.
	backoff []time.Duration
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{},
		backoff: []time.Duration{time.Second, 2 * time.Second},
	}
}

func (c *Client) generate(ctx context.Context, op string, req request) (_ string, err error) {
	defer observe(op, time.Now(), &err)

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("gemini: marshal: %w", err)
	}
	resp, err := c.post(ctx, c.baseURL+"/"+model+":generateContent", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read body: %w", err)
	}
	var res response
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("gemini: HTTP %d: unexpected body: %w", resp.StatusCode, err)
	}
	if res.Error != nil {
		return "", fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, res.Error.Message)
	}
	text := res.text()
	if text == "" {
		return "", errors.New("gemini: empty response")
	}
	return text, nil
}

// post retries network errors, 429 and 5xx with jittered backoff.
func (c *Client) post(ctx context.Context, url string, body []byte) (*http.Response, error) {
	var lastErr error
	var retryAfter time.Duration

	for attempt := 0; attempt <= len(c.backoff); attempt++ {
		if attempt > 0 {
			wait := max(jitter(c.backoff[attempt-1]), retryAfter)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := c.newRequest(ctx, url, body)
		if err != nil {
			return nil, err
		}
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("gemini: %w", err)
			continue
		}
		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		}

		retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
		resp.Body.Close()
		lastErr = fmt.Errorf("gemini: HTTP %d", resp.StatusCode)
	}
	return nil, lastErr
}

func (c *Client) newRequest(ctx context.Context, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)
	return req, nil
}

func buildRequest(system string, history []Message) request {
	contents := make([]content, len(history))
	for i, m := range history {
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents[i] = content{Role: role, Parts: []part{{Text: m.Text}}}
	}
	req := request{
		Contents:         contents,
		GenerationConfig: generationConfig{Temperature: temperature, MaxOutputTokens: maxOutputTokens},
	}
	if system != "" {
		req.SystemInstruction = &content{Parts: []part{{Text: system}}}
	}
	return req
}

// jitter returns a random duration in [d/2, 3d/2).
func jitter(d time.Duration) time.Duration {
	return d/2 + rand.N(d)
}

func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

func observe(op string, start time.Time, err *error) {
	outcome := "ok"
	if *err != nil {
		outcome = "error"
	}
	metrics.LLMRequests.WithLabelValues(op, outcome).Inc()
	metrics.LLMDuration.WithLabelValues(op).Observe(time.Since(start).Seconds())
}
