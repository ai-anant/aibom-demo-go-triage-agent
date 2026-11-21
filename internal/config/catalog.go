package config

// ModelProfile registers one AI model that this service knows how to use.
// Keeping the catalog in one place makes the inventory explicit for reviewers
// and for AIBOM tooling.
type ModelProfile struct {
	ID       string
	Provider string
	Model    string
	Task     string
}

// Profiles is the model registry used to resolve defaults and to advertise
// the service's AI dependencies on the index endpoint.
var Profiles = []ModelProfile{
	{ID: "hf-chat", Provider: "huggingface", Model: "meta-llama/Llama-3.1-8B-Instruct", Task: "chat"},
	{ID: "hf-embedding", Provider: "huggingface", Model: "sentence-transformers/all-MiniLM-L6-v2", Task: "feature-extraction"},
	{ID: "hf-zero-shot", Provider: "huggingface", Model: "facebook/bart-large-mnli", Task: "zero-shot-classification"},
	{ID: "openai-chat", Provider: "openai", Model: "gpt-4o-mini", Task: "chat"},
	{ID: "openai-escalation", Provider: "openai", Model: "gpt-4o", Task: "chat"},
	{ID: "openai-embedding", Provider: "openai", Model: "text-embedding-3-small", Task: "embedding"},
	{ID: "openai-moderation", Provider: "openai", Model: "omni-moderation-latest", Task: "moderation"},
	{ID: "anthropic-chat", Provider: "anthropic", Model: "claude-sonnet-4-5", Task: "chat"},
	{ID: "ollama-chat", Provider: "ollama", Model: "llama3.2", Task: "chat"},
}

// modelFor resolves a profile by identifier, panicking on a bad label so a
// typo fails at startup rather than at request time.
func modelFor(id string) string {
	for _, profile := range Profiles {
		if profile.ID == id {
			return profile.Model
		}
	}
	panic("unknown model profile: " + id)
}
