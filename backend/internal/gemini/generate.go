package gemini

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	return c.generate(ctx, "generate", buildRequest("", []Message{{Role: "user", Text: prompt}}))
}

// GenerateJSON asks for a response matching schema and decodes it into v.
func (c *Client) GenerateJSON(ctx context.Context, prompt string, schema *Schema, v any) error {
	req := buildRequest("", []Message{{Role: "user", Text: prompt}})
	req.GenerationConfig.ResponseMIMEType = "application/json"
	req.GenerationConfig.ResponseSchema = schema

	raw, err := c.generate(ctx, "generate_json", req)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return fmt.Errorf("gemini: decode json response: %w", err)
	}
	return nil
}

// Stream closes both channels when the response ends.
func (c *Client) Stream(ctx context.Context, system string, history []Message) (<-chan string, <-chan error) {
	textCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(textCh)
		defer close(errCh)

		var err error
		defer observe("stream", time.Now(), &err)
		if err = c.stream(ctx, system, history, textCh); err != nil {
			errCh <- err
		}
	}()
	return textCh, errCh
}

func (c *Client) stream(ctx context.Context, system string, history []Message, out chan<- string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	body, err := json.Marshal(buildRequest(system, history))
	if err != nil {
		return fmt.Errorf("gemini: marshal: %w", err)
	}
	req, err := c.newRequest(ctx, c.baseURL+"/"+model+":streamGenerateContent?alt=sse", body)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("gemini: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, raw)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		data, ok := strings.CutPrefix(scanner.Text(), "data: ")
		if !ok {
			continue
		}
		var res response
		if err := json.Unmarshal([]byte(data), &res); err != nil {
			return fmt.Errorf("gemini: parse chunk: %w", err)
		}
		if res.Error != nil {
			return fmt.Errorf("gemini: %s", res.Error.Message)
		}
		if text := res.text(); text != "" {
			select {
			case out <- text:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return scanner.Err()
}
