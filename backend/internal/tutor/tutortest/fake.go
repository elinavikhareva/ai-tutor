// Package tutortest provides a scripted LLM for tests.
package tutortest

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
)

// FakeLLM implements tutor.LLM. Unset responses make the call fail, so a
// test notices when the code talks to the model unexpectedly.
type FakeLLM struct {
	mu      sync.Mutex
	text    []string
	json    []string
	chunks  []string
	err     error
	prompts []string
}

// ReturnText queues responses for Generate.
func (f *FakeLLM) ReturnText(texts ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.text = append(f.text, texts...)
}

// ReturnJSON queues raw JSON responses for GenerateJSON.
func (f *FakeLLM) ReturnJSON(docs ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.json = append(f.json, docs...)
}

// StreamChunks sets the chunks every Stream call produces.
func (f *FakeLLM) StreamChunks(chunks ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chunks = chunks
}

// Fail makes every call return err.
func (f *FakeLLM) Fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *FakeLLM) Prompts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.prompts...)
}

func (f *FakeLLM) Generate(_ context.Context, prompt string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prompts = append(f.prompts, prompt)
	if f.err != nil {
		return "", f.err
	}
	if len(f.text) == 0 {
		return "", errors.New("tutortest: unexpected Generate call")
	}
	text := f.text[0]
	f.text = f.text[1:]
	return text, nil
}

func (f *FakeLLM) GenerateJSON(_ context.Context, prompt string, _ *gemini.Schema, v any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prompts = append(f.prompts, prompt)
	if f.err != nil {
		return f.err
	}
	if len(f.json) == 0 {
		return errors.New("tutortest: unexpected GenerateJSON call")
	}
	doc := f.json[0]
	f.json = f.json[1:]
	return json.Unmarshal([]byte(doc), v)
}

func (f *FakeLLM) Stream(_ context.Context, system string, _ []gemini.Message) (<-chan string, <-chan error) {
	f.mu.Lock()
	f.prompts = append(f.prompts, system)
	chunks, err := f.chunks, f.err
	f.mu.Unlock()

	textCh := make(chan string, len(chunks))
	errCh := make(chan error, 1)
	for _, c := range chunks {
		textCh <- c
	}
	if err != nil {
		errCh <- err
	}
	close(textCh)
	close(errCh)
	return textCh, errCh
}
