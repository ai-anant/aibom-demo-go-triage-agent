// Package llm contains the chat-model clients used by the triage service.
//
// Hosted providers are wrapped behind the small Client interface so the
// classifier can be tested without network access.
package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
)

// Client is the minimal surface the triage service needs from a model.
type Client interface {
	// Complete returns the model's raw text response.
	Complete(ctx context.Context, system, user string) (string, error)
	// Name reports the model identifier used for this client.
	Name() string
}

// New builds the client for the configured provider.
func New(cfg config.Config) (Client, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAI(cfg)
	case "anthropic":
		return NewAnthropic(cfg)
	case "ollama":
		return NewOllama(cfg), nil
	default:
		return NewStub(), nil
	}
}

// Stub is a deterministic offline client used for demos and tests. It returns
// a conservative classification that the caller can override with rule-based
// routing.
type Stub struct{}

// NewStub returns the offline client.
func NewStub() *Stub { return &Stub{} }

// Name implements Client.
func (s *Stub) Name() string { return "stub-rules" }

// Complete implements Client.
func (s *Stub) Complete(_ context.Context, _ string, user string) (string, error) {
	priority := "normal"
	queue := "general"
	lowered := strings.ToLower(ticketSection(user))
	switch {
	case strings.Contains(lowered, "breach"), strings.Contains(lowered, "credential"),
		strings.Contains(lowered, "api key"), strings.Contains(lowered, "leak"):
		priority = "urgent"
		queue = "security"
	case strings.Contains(lowered, "outage"), strings.Contains(lowered, "down"),
		strings.Contains(lowered, "urgent"):
		priority = "urgent"
		queue = "incident-response"
	case strings.Contains(lowered, "refund"), strings.Contains(lowered, "invoice"),
		strings.Contains(lowered, "billing"):
		queue = "billing"
	case strings.Contains(lowered, "password"), strings.Contains(lowered, "login"):
		queue = "account"
	}

	return fmt.Sprintf(
		`{"priority":%q,"queue":%q,"confidence":0.55,"rationale":"offline keyword rules","escalate":%t}`,
		priority, queue, priority == "urgent",
	), nil
}

// ticketSection trims the retrieved policy context so keyword rules only look
// at the ticket itself, not at policy text that mentions every category.
func ticketSection(prompt string) string {
	if idx := strings.Index(prompt, "Relevant policy notes:"); idx >= 0 {
		return prompt[:idx]
	}
	return prompt
}
