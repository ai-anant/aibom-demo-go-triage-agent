package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/sashabaranov/go-openai"
)

// OpenAI is the hosted chat client used in production.
type OpenAI struct {
	client *openai.Client
	model  string
}

// NewOpenAI builds a client for the OpenAI chat completions API.
func NewOpenAI(cfg config.Config) (*OpenAI, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for the openai provider")
	}
	return &OpenAI{
		client: openai.NewClient(apiKey),
		model:  cfg.ChatModel,
	}, nil
}

// Name implements Client.
func (o *OpenAI) Name() string { return o.model }

// Complete implements Client using JSON-mode chat completions.
func (o *OpenAI) Complete(ctx context.Context, system, user string) (string, error) {
	resp, err := o.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       o.model,
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
		return "", fmt.Errorf("openai chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}

// Embed returns a vector for the configured embedding model. Used for the
// optional semantic re-rank of knowledge-base hits.
func (o *OpenAI) Embed(ctx context.Context, text string, model string) ([]float32, error) {
	resp, err := o.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(model),
		Input: []string{text},
	})
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai returned no embeddings")
	}
	return resp.Data[0].Embedding, nil
}
