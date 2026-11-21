package hf

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbedParsesBatchResponse(t *testing.T) {
	var gotPath, gotAuth, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("authorization")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		_, _ = w.Write([]byte(`[[0.1,0.2],[0.3,0.4]]`))
	}))
	defer server.Close()

	client := New("hf_test_token", server.URL)
	vectors, err := client.Embed(
		context.Background(), "sentence-transformers/all-MiniLM-L6-v2", []string{"a", "b"},
	)
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vectors) != 2 || vectors[1][1] != 0.4 {
		t.Fatalf("unexpected vectors: %v", vectors)
	}
	if gotPath != "/sentence-transformers/all-MiniLM-L6-v2" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer hf_test_token" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"inputs":["a","b"]`) {
		t.Fatalf("body = %s", gotBody)
	}
}

func TestZeroShotParsesLabelsAndScores(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			`{"labels":["urgent security incident or outage","routine request"],"scores":[0.88,0.12]}`,
		))
	}))
	defer server.Close()

	client := New("", server.URL)
	scorer := NewScorer(client, "facebook/bart-large-mnli")

	score, err := scorer.UrgencyScore(context.Background(), "All requests are failing")
	if err != nil {
		t.Fatalf("urgency: %v", err)
	}
	if score != 0.88 {
		t.Fatalf("score = %v", score)
	}
}

func TestZeroShotHandlesReorderedLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(
			`{"labels":["routine request","urgent security incident or outage"],"scores":[0.3,0.7]}`,
		))
	}))
	defer server.Close()

	score, err := NewScorer(New("", server.URL), "facebook/bart-large-mnli").
		UrgencyScore(context.Background(), "password reset please")
	if err != nil {
		t.Fatalf("urgency: %v", err)
	}
	if score != 0.7 {
		t.Fatalf("score = %v", score)
	}
}

func TestNonOKStatusIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"model loading"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := New("", server.URL).Embed(context.Background(), "m", []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected 503 error, got %v", err)
	}
}
