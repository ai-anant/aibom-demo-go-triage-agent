package hf

import (
	"context"
	"fmt"
)

// UrgentLabel is the positive class used when scoring a ticket.
const UrgentLabel = "urgent security incident or outage"

// Scorer turns a ticket into an urgency score with a zero-shot classifier,
// giving the triage service a second opinion independent of the chat model.
type Scorer struct {
	client *Client
	model  string
}

// NewScorer builds a zero-shot urgency scorer.
func NewScorer(client *Client, model string) *Scorer {
	return &Scorer{client: client, model: model}
}

// UrgencyScore returns the probability that the ticket needs escalation.
func (s *Scorer) UrgencyScore(ctx context.Context, text string) (float64, error) {
	results, err := s.client.ZeroShot(
		ctx, s.model, text, []string{UrgentLabel, "routine request"},
	)
	if err != nil {
		return 0, err
	}
	for _, result := range results {
		if result.Label == UrgentLabel {
			return result.Score, nil
		}
	}
	return 0, fmt.Errorf("zero-shot response did not contain %q", UrgentLabel)
}
