# Triage Agent — Go AI sample for AIBOM demos

A support-ticket triage service that classifies inbound tickets with an LLM,
routes them to the right queue, and escalates risky cases to a human. It is the
Go counterpart to the Python `policy-assistant` sample and exists to show an
**AI Bill of Materials (AIBOM)** scan over a Go codebase.

- **Language / runtime:** Go 1.24+, `net/http`, `log/slog`
- **Models declared:** `gpt-4o-mini`, `gpt-4o`, `claude-sonnet-4-5`,
  `llama3.2` (local Ollama), `text-embedding-3-small`, `omni-moderation-latest`
- **Frameworks:** OpenAI Go SDK (`sashabaranov/go-openai`),
  LangChainGo (`tmc/langchaingo`) for the escalation summary chain
- **Offline by default:** `AI_PROVIDER=stub` returns deterministic
  rule-based classifications, so the demo and tests need no credentials.

> Companion sample: [`aibom-demo-python-policy-assistant`](https://github.com/ai-anant/aibom-demo-python-policy-assistant) —
> the same idea implemented in Python, useful for comparing per-language scan depth.

## Flow

```
POST /v1/tickets
   │
   ├── kb.Search            embedded policy notes -> prompt context
   ├── prompts/*.txt        system + user prompt templates
   ├── llm.Client           openai | anthropic | ollama | stub
   ├── triage.Classify      JSON validation, escalation rules
   └── store.Add            in-memory record, id TRG-0001
```

Escalation rules that always apply, regardless of model confidence:

* `security` queue tickets are escalated;
* Enterprise tickets marked `urgent` are escalated;
* any classification with confidence below `0.5` is escalated.

When the model call fails or returns non-JSON, the service falls back to
deterministic keyword rules and flags the result with `"fallback": true`.

## Quick start (no API key)

```bash
go build ./...
go test ./...

AI_PROVIDER=stub go run ./cmd/triage-agent
```

```bash
curl -s localhost:8080/healthz
curl -s -X POST localhost:8080/v1/tickets \
  -H 'content-type: application/json' \
  -d '{"subject":"Possible credential leak","body":"A contractor pasted an API key into a public gist.","tier":"enterprise"}'
```

## Providers

```bash
# OpenAI (default when OPENAI_API_KEY is set)
export AI_PROVIDER=openai OPENAI_API_KEY=sk-...
go run ./cmd/triage-agent

# Anthropic via LangChainGo
export AI_PROVIDER=anthropic ANTHROPIC_API_KEY=sk-ant-...
go run ./cmd/triage-agent

# Local Ollama (no cloud dependency)
ollama pull llama3.2
export AI_PROVIDER=ollama OLLAMA_MODEL=llama3.2
go run ./cmd/triage-agent
```

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/healthz` | liveness plus active provider/model |
| `GET` | `/` | service metadata, model catalog |
| `POST` | `/v1/tickets` | classify and store a ticket |
| `GET` | `/v1/tickets` | list stored tickets |
| `GET` | `/v1/tickets/{id}` | fetch one ticket |

## Layout

```
cmd/triage-agent/        entrypoint and graceful shutdown
internal/config/         settings + model catalog constants
internal/llm/            Client interface, OpenAI, Anthropic, Ollama, stub,
                         LangChain summarizer
internal/kb/             embedded policy notes + TF-IDF retrieval
internal/prompts/        embedded prompt templates
internal/triage/         classification, validation, escalation, rules
internal/store/          in-memory ticket store
internal/httpapi/        HTTP handlers
docs/AIBOM_DEMO.md       how to produce an AIBOM from this repo
```

## Generating an AIBOM

```bash
uv tool install --python 3.13 "cisco-aibom[agentic,llm-openai]"
cisco-aibom analyze . -o json -O aibom-report.json --llm-model gpt-4o-mini
```

For Go, the scanner reads `go.mod` for AI dependencies and matches model
literals in source. Details and the expected inventory are in
[docs/AIBOM_DEMO.md](docs/AIBOM_DEMO.md).

## Design notes for the demo

* Model identifiers are constants in `internal/config/config.go`, so the scan
  resolves them consistently across clients.
* Prompt text lives in `internal/prompts/*.txt` and is embedded with
  `go:embed`, which mirrors how real Go services ship prompts.
* The `Client` interface keeps hosted providers swappable and lets tests run
  without network access.
* `Dockerfile` builds a static binary; the image can be scanned as a container
  instead of a source tree.

## License

MIT
