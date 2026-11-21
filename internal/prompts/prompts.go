// Package prompts embeds the prompt templates used by the triage service.
package prompts

import (
	"embed"
	"fmt"
)

//go:embed *.txt
var files embed.FS

// Load returns the named prompt template from the embedded bundle.
func Load(name string) (string, error) {
	data, err := files.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("prompt %q: %w", name, err)
	}
	return string(data), nil
}
