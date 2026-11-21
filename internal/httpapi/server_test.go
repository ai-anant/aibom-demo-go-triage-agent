package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/kb"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/llm"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/store"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/triage"
)

func newServer(t *testing.T) *Server {
	t.Helper()
	base, err := kb.Load()
	if err != nil {
		t.Fatalf("load kb: %v", err)
	}
	classifier, err := triage.NewClassifier(llm.NewStub(), base, 3)
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}
	cfg := config.FromEnv()
	cfg.Provider = "stub"
	return New(cfg, classifier, store.New())
}

func TestCreateAndFetchTicket(t *testing.T) {
	handler := newServer(t).Routes()

	body, _ := json.Marshal(map[string]string{
		"subject": "Complete outage in eu-west",
		"body":    "All requests are failing since 09:10 UTC.",
		"tier":    "enterprise",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/tickets", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	var created struct {
		ID     string        `json:"id"`
		Result triage.Result `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ID != "TRG-0001" {
		t.Fatalf("id = %q", created.ID)
	}
	if created.Result.Queue != "incident-response" {
		t.Fatalf("queue = %q", created.Result.Queue)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/tickets/"+created.ID, nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch status = %d", rec.Code)
	}
}

func TestCreateRejectsMissingFields(t *testing.T) {
	handler := newServer(t).Routes()
	req := httptest.NewRequest(http.MethodPost, "/v1/tickets",
		bytes.NewReader([]byte(`{"subject":"only subject"}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHealthAndIndex(t *testing.T) {
	handler := newServer(t).Routes()

	for _, path := range []string{"/healthz", "/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
	}
}

func TestUnknownTicketReturns404(t *testing.T) {
	handler := newServer(t).Routes()
	req := httptest.NewRequest(http.MethodGet, "/v1/tickets/TRG-9999", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
