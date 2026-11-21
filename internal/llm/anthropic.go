package llm

import (
	"context"
	"fmt"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
)

// Anthropic wraps langchaingo's Anthropic provider, used as the secondary
// reviewer for escalations.
type Anthropic struct {
	model string
	llm   llms.Model
}

// NewAnthropic builds a Claude client from the environment.
func NewAnthropic(cfg config.Config) (*Anthropic, error) {
	model, err := anthropic.New(anthropic.WithModel(cfg.AnthropicModel))
	if err != nil {
		return nil, fmt.Errorf("anthropic client: %w", err)
	}
	return &Anthropic{model: cfg.AnthropicModel, llm: model}, nil
}

// Name implements Client.
func (a *Anthropic) Name() string { return a.model }

// Complete implements Client.
func (a *Anthropic) Complete(ctx context.Context, system, user string) (string, error) {
	prompt := fmt.Sprintf("%s\n\n%s", system, user)
	out, err := a.llm.Call(ctx, prompt, llms.WithTemperature(0.1), llms.WithMaxTokens(700))
	if err != nil {
		return "", fmt.Errorf("anthropic completion: %w", err)
	}
	return out, nil
}
