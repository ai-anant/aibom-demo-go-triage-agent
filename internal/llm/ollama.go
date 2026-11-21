package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
)

// Ollama talks to a local Ollama runtime, used for air-gapped demos.
type Ollama struct {
	baseURL string
	model   string
	http    *http.Client
}

// NewOllama builds a client for the local Ollama HTTP API.
func NewOllama(cfg config.Config) *Ollama {
	return &Ollama{
		baseURL: cfg.OllamaBaseURL,
		model:   cfg.OllamaModel,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Name implements Client.
func (o *Ollama) Name() string { return o.model }

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

// Complete implements Client.
func (o *Ollama) Complete(ctx context.Context, system, user string) (string, error) {
	payload, err := json.Marshal(ollamaRequest{
		Model:  o.model,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Options: map[string]any{"temperature": 0.1},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(payload),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := o.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %s", resp.Status)
	}

	var out ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ollama decode: %w", err)
	}
	return out.Message.Content, nil
}
