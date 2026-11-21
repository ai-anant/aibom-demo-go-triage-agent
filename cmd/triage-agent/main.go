// Command triage-agent runs the support-ticket triage service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/config"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/hf"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/httpapi"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/kb"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/llm"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/prompts"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/store"
	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/triage"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	_ = godotenv.Load()
	cfg := config.FromEnv()

	base, err := kb.Load()
	if err != nil {
		slog.Error("load knowledge base", "error", err)
		os.Exit(1)
	}

	client, err := llm.New(cfg)
	if err != nil {
		slog.Error("build model client", "error", err)
		os.Exit(1)
	}

	classifier, err := triage.NewClassifier(client, base, cfg.TopK)
	if err != nil {
		slog.Error("build classifier", "error", err)
		os.Exit(1)
	}

	if cfg.Provider == "openai" {
		if template, err := prompts.Load("escalation_summary_prompt.txt"); err == nil {
			if summarizer, err := llm.NewLangChainSummarizer(cfg, template); err == nil {
				classifier.SetSummarizer(summarizer)
			} else {
				slog.Warn("escalation summariser unavailable", "error", err)
			}
		}
	}

	if cfg.HFToken != "" {
		scorer := hf.NewScorer(hf.New(cfg.HFToken, cfg.HFInferenceURL), cfg.HFZeroShotModel)
		classifier.SetUrgencyScorer(scorer)
		slog.Info("hugging face urgency scorer enabled", "model", cfg.HFZeroShotModel)
	}

	tickets := store.New()
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.New(cfg, classifier, tickets).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.Addr, "provider", cfg.Provider, "model", client.Name())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Warn("shutdown", "error", err)
	}
	slog.Info("stopped")
}
