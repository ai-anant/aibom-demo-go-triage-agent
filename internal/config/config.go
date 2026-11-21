// Package config holds runtime configuration and the model catalog.
//
// Model identifiers are declared here so an AIBOM scan can resolve them from a
// single place instead of hunting through string literals in handlers.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Model catalog used by this service.
const (
	OpenAIChatModel       = "gpt-4o-mini"
	OpenAIEscalationModel = "gpt-4o"
	AnthropicChatModel    = "claude-sonnet-4-5"
	OllamaChatModel       = "llama3.2"
	OpenAIEmbeddingModel  = "text-embedding-3-small"
	ModerationModel       = "omni-moderation-latest"
)

// Config is the resolved configuration for one process.
type Config struct {
	Addr            string
	Provider        string
	ChatModel       string
	EscalationModel string
	AnthropicModel  string
	OllamaModel     string
	OllamaBaseURL   string
	EmbeddingModel  string
	Temperature     float32
	MaxTokens       int
	TopK            int
}

// FromEnv resolves configuration from environment variables.
func FromEnv() Config {
	provider := strings.ToLower(os.Getenv("AI_PROVIDER"))
	if provider == "" {
		switch {
		case os.Getenv("OPENAI_API_KEY") != "":
			provider = "openai"
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			provider = "anthropic"
		default:
			provider = "stub"
		}
	}

	return Config{
		Addr:            env("ADDR", ":8080"),
		Provider:        provider,
		ChatModel:       env("CHAT_MODEL", OpenAIChatModel),
		EscalationModel: env("ESCALATION_MODEL", OpenAIEscalationModel),
		AnthropicModel:  env("ANTHROPIC_MODEL", AnthropicChatModel),
		OllamaModel:     env("OLLAMA_MODEL", OllamaChatModel),
		OllamaBaseURL:   env("OLLAMA_BASE_URL", "http://localhost:11434"),
		EmbeddingModel:  env("EMBEDDING_MODEL", OpenAIEmbeddingModel),
		Temperature:     float32(envFloat("TEMPERATURE", 0.1)),
		MaxTokens:       envInt("MAX_TOKENS", 700),
		TopK:            envInt("TOP_K", 3),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(os.Getenv(key), 64)
	if err != nil {
		return fallback
	}
	return value
}
