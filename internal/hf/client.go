// Package hf calls the Hugging Face serverless inference API for embeddings
// and zero-shot classification.
package hf

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LabelScore is one label/score pair returned by a classification task.
type LabelScore struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// Client talks to the inference API.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// New returns a client for the given inference base URL.
func New(token, baseURL string) *Client {
	return &Client{
		token:   token,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Embed calls a feature-extraction model and returns one vector per input.
func (c *Client) Embed(ctx context.Context, model string, inputs []string) ([][]float64, error) {
	var raw json.RawMessage
	if err := c.post(ctx, model, map[string]any{"inputs": inputs}, &raw); err != nil {
		return nil, err
	}

	var batch [][]float64
	if err := json.Unmarshal(raw, &batch); err == nil {
		return batch, nil
	}

	var single []float64
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	return [][]float64{single}, nil
}

// ZeroShot classifies text against candidate labels.
func (c *Client) ZeroShot(
	ctx context.Context, model, text string, labels []string,
) ([]LabelScore, error) {
	payload := map[string]any{
		"inputs":     text,
		"parameters": map[string]any{"candidate_labels": labels},
	}

	var out struct {
		Labels []string  `json:"labels"`
		Scores []float64 `json:"scores"`
	}
	if err := c.post(ctx, model, payload, &out); err != nil {
		return nil, err
	}
	if len(out.Labels) != len(out.Scores) {
		return nil, fmt.Errorf("zero-shot response mismatch: %d labels, %d scores", len(out.Labels), len(out.Scores))
	}

	results := make([]LabelScore, 0, len(out.Labels))
	for i, label := range out.Labels {
		results = append(results, LabelScore{Label: label, Score: out.Scores[i]})
	}
	return results, nil
}

func (c *Client) post(
	ctx context.Context, model string, payload any, out any,
) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/"+model, bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	if c.token != "" {
		req.Header.Set("authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("hf request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("hf returned %s: %s", resp.Status, string(detail))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
