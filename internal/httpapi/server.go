// Package httpapi exposes the triage service over HTTP.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/store"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/triage"
)

// Server holds the HTTP dependencies.
type Server struct {
	cfg        config.Config
	classifier *triage.Classifier
	store      *store.Store
}

// New builds the HTTP server.
func New(cfg config.Config, classifier *triage.Classifier, tickets *store.Store) *Server {
	return &Server{cfg: cfg, classifier: classifier, store: tickets}
}

// Routes returns the mux with all handlers registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("POST /v1/tickets", s.createTicket)
	mux.HandleFunc("GET /v1/tickets", s.listTickets)
	mux.HandleFunc("GET /v1/tickets/{id}", s.getTicket)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"provider": s.cfg.Provider,
		"model":    s.classifier.ModelName(),
	})
}

func (s *Server) index(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service":          "triage-agent",
		"provider":         s.cfg.Provider,
		"chat_model":       s.cfg.ChatModel,
		"escalation_model": s.cfg.EscalationModel,
		"embedding_model":  s.cfg.EmbeddingModel,
		"ollama_model":     s.cfg.OllamaModel,
		"huggingface": map[string]string{
			"chat":      s.cfg.HFChatModel,
			"embedding": s.cfg.HFEmbeddingModel,
			"zero_shot": s.cfg.HFZeroShotModel,
		},
		"hf_enabled":    s.cfg.HFToken != "",
		"model_catalog": config.Profiles,
	})
}

type createRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Tier    string `json:"tier"`
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Subject == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "subject and body are required")
		return
	}
	if req.Tier == "" {
		req.Tier = "starter"
	}

	ticket := triage.Ticket{Subject: req.Subject, Body: req.Body, Tier: req.Tier}
	result := s.classifier.Classify(r.Context(), ticket)
	record := s.store.Add(ticket, result)

	slog.Info("ticket classified",
		"id", record.ID, "queue", result.Queue, "priority", result.Priority,
		"escalate", result.Escalate, "model", result.Model,
	)
	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) listTickets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tickets": s.store.List(),
		"count":   len(s.store.List()),
	})
}

func (s *Server) getTicket(w http.ResponseWriter, r *http.Request) {
	record, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
