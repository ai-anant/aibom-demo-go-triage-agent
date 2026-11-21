// Package triage classifies inbound support tickets with an LLM and falls back
// to deterministic rules when the model is unavailable or returns garbage.
package triage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"text/template"
	"time"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/kb"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/llm"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/prompts"
)

var allowedPriorities = map[string]bool{
	"urgent": true, "high": true, "normal": true, "low": true,
}

var allowedQueues = map[string]bool{
	"incident-response": true, "security": true, "billing": true,
	"account": true, "general": true,
}

// Ticket is the inbound support request.
type Ticket struct {
	ID      string    `json:"id"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	Tier    string    `json:"tier"`
	Created time.Time `json:"created"`
}

// Result is the classification outcome returned to the caller.
type Result struct {
	Priority   string  `json:"priority"`
	Queue      string  `json:"queue"`
	Confidence float64 `json:"confidence"`
	Rationale  string  `json:"rationale"`
	Escalate   bool    `json:"escalate"`
	Summary    string  `json:"summary,omitempty"`
	Model      string  `json:"model"`
	Fallback   bool    `json:"fallback"`
}

// Classifier couples a chat model with the policy knowledge base.
type Classifier struct {
	client      llm.Client
	base        *kb.Base
	topK        int
	system      string
	userTpl     *template.Template
	summarizer  llm.Summarizer
	escalations int
}

// NewClassifier loads the prompt templates and prepares a classifier.
func NewClassifier(client llm.Client, base *kb.Base, topK int) (*Classifier, error) {
	system, err := prompts.Load("classify_system_prompt.txt")
	if err != nil {
		return nil, err
	}
	raw, err := prompts.Load("classify_user_prompt.txt")
	if err != nil {
		return nil, err
	}
	tpl, err := template.New("classify").Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse classify template: %w", err)
	}
	if topK <= 0 {
		topK = 3
	}
	return &Classifier{
		client:  client,
		base:    base,
		topK:    topK,
		system:  system,
		userTpl: tpl,
	}, nil
}

// SetSummarizer attaches an optional escalation summariser.
func (c *Classifier) SetSummarizer(s llm.Summarizer) { c.summarizer = s }

// ModelName reports the model identifier backing the classifier.
func (c *Classifier) ModelName() string { return c.client.Name() }

// Classify runs the model, validates its answer and applies escalation rules.
func (c *Classifier) Classify(ctx context.Context, ticket Ticket) Result {
	verdict, err := c.askModel(ctx, ticket)
	if err != nil {
		slog.Warn("model classification failed, using rules", "error", err)
		verdict = rules(ticket)
		verdict.Fallback = true
	}
	verdict.Model = c.client.Name()

	if !allowedPriorities[verdict.Priority] {
		verdict.Priority = "normal"
	}
	if !allowedQueues[verdict.Queue] {
		verdict.Queue = "general"
	}
	if verdict.Confidence < 0.5 {
		verdict.Escalate = true
	}
	if strings.EqualFold(ticket.Tier, "enterprise") && verdict.Priority == "urgent" {
		verdict.Escalate = true
	}
	if verdict.Queue == "security" {
		verdict.Escalate = true
	}
	if verdict.Escalate {
		verdict.Summary = c.summarize(ctx, ticket)
	}
	return verdict
}

func (c *Classifier) askModel(ctx context.Context, ticket Ticket) (Result, error) {
	hits := c.base.Search(ticket.Subject+" "+ticket.Body, c.topK)
	contextText := "(no matching policy notes)"
	if len(hits) > 0 {
		parts := make([]string, 0, len(hits))
		for _, hit := range hits {
			parts = append(parts, fmt.Sprintf("[%s] %s", hit.ID, hit.Body))
		}
		contextText = strings.Join(parts, "\n\n")
	}

	var prompt strings.Builder
	err := c.userTpl.Execute(&prompt, map[string]string{
		"tier":    ticket.Tier,
		"subject": ticket.Subject,
		"body":    ticket.Body,
		"context": contextText,
	})
	if err != nil {
		return Result{}, fmt.Errorf("render prompt: %w", err)
	}

	raw, err := c.client.Complete(ctx, c.system, prompt.String())
	if err != nil {
		return Result{}, err
	}

	payload := extractJSON(raw)
	var verdict Result
	if err := json.Unmarshal([]byte(payload), &verdict); err != nil {
		return Result{}, fmt.Errorf("decode model json: %w", err)
	}
	return verdict, nil
}

func (c *Classifier) summarize(ctx context.Context, ticket Ticket) string {
	if c.summarizer == nil {
		return fmt.Sprintf("%s: %s", ticket.Subject, truncate(ticket.Body, 200))
	}
	summary, err := c.summarizer.Summarize(
		ctx, fmt.Sprintf("%s\n%s", ticket.Subject, ticket.Body),
	)
	if err != nil {
		slog.Warn("escalation summariser failed", "error", err)
		return fmt.Sprintf("%s: %s", ticket.Subject, truncate(ticket.Body, 200))
	}
	return strings.TrimSpace(summary)
}

// rules is the deterministic fallback used when the model is unavailable.
func rules(ticket Ticket) Result {
	text := strings.ToLower(ticket.Subject + " " + ticket.Body)
	verdict := Result{
		Priority:   "normal",
		Queue:      "general",
		Confidence: 0.4,
		Rationale:  "rule-based routing: no model response",
	}

	switch {
	case containsAny(text, "breach", "leak", "unauthorised access", "unauthorized access", "credential"):
		verdict.Priority, verdict.Queue = "urgent", "security"
	case containsAny(text, "outage", "down", "data loss", "cannot process payments"):
		verdict.Priority, verdict.Queue = "urgent", "incident-response"
	case containsAny(text, "refund", "invoice", "billing", "charge"):
		verdict.Priority, verdict.Queue = "high", "billing"
	case containsAny(text, "password", "login", "sso", "seat"):
		verdict.Queue = "account"
	}

	if verdict.Priority == "urgent" || strings.EqualFold(ticket.Tier, "enterprise") {
		verdict.Escalate = true
	}
	return verdict
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func extractJSON(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}
