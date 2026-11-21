package triage

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/kb"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/llm"
)

type fakeClient struct {
	response string
	err      error
}

func (f fakeClient) Complete(context.Context, string, string) (string, error) {
	return f.response, f.err
}

func (f fakeClient) Name() string { return "fake-model" }

func newClassifier(t *testing.T, client llm.Client) *Classifier {
	t.Helper()
	base, err := kb.Load()
	if err != nil {
		t.Fatalf("load kb: %v", err)
	}
	c, err := NewClassifier(client, base, 3)
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}
	return c
}

func TestClassifyParsesModelJSON(t *testing.T) {
	client := fakeClient{response: "```json\n" +
		`{"priority":"urgent","queue":"security","confidence":0.93,"rationale":"credential leak","escalate":false}` +
		"\n```"}
	c := newClassifier(t, client)

	got := c.Classify(context.Background(), Ticket{
		Subject: "Possible credential leak",
		Body:    "A contractor pasted an API key into a public gist.",
		Tier:    "enterprise",
	})

	if got.Priority != "urgent" || got.Queue != "security" {
		t.Fatalf("unexpected verdict: %+v", got)
	}
	if !got.Escalate {
		t.Fatal("security queue must always escalate")
	}
	if got.Model != "fake-model" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.Summary == "" {
		t.Fatal("expected a hand-off summary")
	}
}

func TestClassifyFallsBackToRulesOnError(t *testing.T) {
	c := newClassifier(t, fakeClient{err: errors.New("upstream 503")})

	got := c.Classify(context.Background(), Ticket{
		Subject: "Refund for duplicate charge",
		Body:    "We were billed twice for the October invoice.",
		Tier:    "business",
	})

	if !got.Fallback {
		t.Fatal("expected fallback flag")
	}
	if got.Queue != "billing" || got.Priority != "high" {
		t.Fatalf("unexpected rules verdict: %+v", got)
	}
}

func TestClassifyFallsBackOnInvalidJSON(t *testing.T) {
	c := newClassifier(t, fakeClient{response: "I think this is urgent?"})

	got := c.Classify(context.Background(), Ticket{
		Subject: "Site is down",
		Body:    "Complete outage for all users.",
		Tier:    "starter",
	})

	if !got.Fallback {
		t.Fatal("expected fallback flag")
	}
	if got.Queue != "incident-response" || got.Priority != "urgent" {
		t.Fatalf("unexpected rules verdict: %+v", got)
	}
}

func TestUnknownValuesAreSanitized(t *testing.T) {
	c := newClassifier(t, fakeClient{response: `{"priority":"catastrophic","queue":"vibes","confidence":0.8}`})

	got := c.Classify(context.Background(), Ticket{
		Subject: "Question about invoicing",
		Body:    "Where can I download last month's invoice?",
		Tier:    "starter",
	})

	if got.Priority != "normal" || got.Queue != "general" {
		t.Fatalf("unexpected sanitized verdict: %+v", got)
	}
}

func TestLowConfidenceEscalates(t *testing.T) {
	c := newClassifier(t, fakeClient{response: `{"priority":"normal","queue":"account","confidence":0.3}`})

	got := c.Classify(context.Background(), Ticket{
		Subject: "SSO login loop",
		Body:    "Users bounce between the IdP and the app.",
		Tier:    "business",
	})

	if !got.Escalate {
		t.Fatal("low confidence should escalate")
	}
}

type fakeScorer struct {
	score float64
	err   error
}

func (f fakeScorer) UrgencyScore(context.Context, string) (float64, error) {
	return f.score, f.err
}

func TestUrgencyScorerForcesEscalation(t *testing.T) {
	c := newClassifier(t, fakeClient{response: `{"priority":"normal","queue":"account","confidence":0.9}`})
	c.SetUrgencyScorer(fakeScorer{score: 0.91})

	got := c.Classify(context.Background(), Ticket{
		Subject: "Password reset",
		Body:    "Please reset my password.",
		Tier:    "starter",
	})

	if !got.Escalate {
		t.Fatal("high HF urgency score should escalate")
	}
	if got.UrgencyScore != 0.91 {
		t.Fatalf("urgency score = %v", got.UrgencyScore)
	}
}

func TestUrgencyScorerErrorKeepsVerdict(t *testing.T) {
	c := newClassifier(t, fakeClient{response: `{"priority":"low","queue":"general","confidence":0.9}`})
	c.SetUrgencyScorer(fakeScorer{err: errors.New("model loading")})

	got := c.Classify(context.Background(), Ticket{
		Subject: "Documentation link",
		Body:    "Where is the API reference?",
		Tier:    "starter",
	})

	if got.Escalate {
		t.Fatal("scorer failure must not force escalation")
	}
}
