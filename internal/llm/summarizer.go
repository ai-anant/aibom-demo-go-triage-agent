package llm

import (
	"context"
	"fmt"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

// Summarizer condenses a ticket into a hand-off note for a human queue.
type Summarizer interface {
	Summarize(ctx context.Context, ticket string) (string, error)
}

// LangChainSummarizer builds a one-step LangChain LLMChain over the
// escalation prompt template.
type LangChainSummarizer struct {
	chain chains.Chain
}

// NewLangChainSummarizer wires the escalation prompt into a LangChain chain.
func NewLangChainSummarizer(cfg config.Config, template string) (*LangChainSummarizer, error) {
	model, err := openai.New(openai.WithModel(cfg.EscalationModel))
	if err != nil {
		return nil, fmt.Errorf("langchain openai llm: %w", err)
	}

	prompt := prompts.NewPromptTemplate(template, []string{"ticket"})
	chain := chains.NewLLMChain(model, prompt)
	return &LangChainSummarizer{chain: chain}, nil
}

// Summarize implements Summarizer.
func (s *LangChainSummarizer) Summarize(ctx context.Context, ticket string) (string, error) {
	out, err := chains.Run(ctx, s.chain, map[string]any{"ticket": ticket})
	if err != nil {
		return "", fmt.Errorf("langchain summarize: %w", err)
	}
	return out, nil
}
