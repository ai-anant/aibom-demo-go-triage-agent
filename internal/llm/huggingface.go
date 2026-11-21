package llm

import (
	"context"
	"fmt"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/sashabaranov/go-openai"
)

// HuggingFace talks to the Hugging Face router, which exposes an
// OpenAI-compatible chat completions API.
type HuggingFace struct {
	client *openai.Client
	model  string
}

// NewHuggingFace builds a chat client for the HF router.
func NewHuggingFace(cfg config.Config) (*HuggingFace, error) {
	if cfg.HFToken == "" {
		return nil, fmt.Errorf("HF_TOKEN is required for the huggingface provider")
	}

	conf := openai.DefaultConfig(cfg.HFToken)
	conf.BaseURL = cfg.HFChatBaseURL
	return &HuggingFace{
		client: openai.NewClientWithConfig(conf),
		model:  cfg.HFChatModel,
	}, nil
}

// Name implements Client.
func (h *HuggingFace) Name() string { return h.model }

// Complete implements Client using JSON-mode chat completions.
func (h *HuggingFace) Complete(ctx context.Context, system, user string) (string, error) {
	resp, err := h.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       h.model,
		Temperature: 0.1,
		MaxTokens:   700,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
	})
	if err != nil {
		return "", fmt.Errorf("huggingface chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("huggingface returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
