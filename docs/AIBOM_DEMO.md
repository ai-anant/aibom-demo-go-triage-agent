# Generating an AIBOM from this repository (Go)

This repository is a Go target for AI Bill of Materials tooling. Go is a
second-tier language for most AIBOM scanners: they resolve dependencies from
`go.mod`, pull model identifiers out of source literals, and rely on the
LLM-assisted pass for confirmation.

## 1. Install a scanner

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
uv tool install --python 3.13 "cisco-aibom[agentic,llm-openai]"
export OPENAI_API_KEY=sk-...
```

## 2. Scan

```bash
cd aibom-demo-go-triage-agent
cisco-aibom analyze . \
  --output-format json \
  --output-file aibom-report.json \
  --llm-model gpt-4o-mini
```

Variants worth showing in a demo:

```bash
cisco-aibom analyze . -o plaintext
cisco-aibom analyze . -o cyclonedx -O aibom.cdx.json --llm-model gpt-4o-mini
cisco-aibom analyze . -o html -O aibom-report.html --llm-model gpt-4o-mini
```

## 3. What to expect in the report

| Asset type | Where it comes from |
|---|---|
| `model` | `internal/config/catalog.go` registry, `internal/llm/*.go` |
| `model` (Hugging Face) | `catalog.go` + `internal/hf/` — `Llama-3.1-8B-Instruct`, `all-MiniLM-L6-v2`, `bart-large-mnli` |
| `llm_endpoint` | `internal/llm/openai.go`, `huggingface.go`, `anthropic.go`, `ollama.go` |
| `embedding` | `text-embedding-3-small` in `internal/llm/openai.go` |
| `prompt` | `internal/prompts/classify_system_prompt.txt`, `classify_user_prompt.txt`, `escalation_summary_prompt.txt` (surfaced by the agentic pass) |
| `agent` | `internal/triage/triage.go` classification loop with tool-less completion (framework-level agent evidence comes from LangChainGo usage) |
| `dependency` | `go.mod`: `go-openai`, `langchaingo` |
| `dataset` | not used by this sample; the policy notes are local Markdown |

## 4. Verified deterministic output

Running only the deterministic stage (Cisco AIBOM 1.10.1, no LLM) on this
repository reports **29 components**:

* dependency entries for `github.com/sashabaranov/go-openai` and
  `github.com/tmc/langchaingo` (from `go.mod` and imports)
* models `gpt-4o-mini`, `gpt-4o`, `claude-sonnet-4-5`, `llama3.2` resolved from
  the model registry
* Hugging Face models `meta-llama/Llama-3.1-8B-Instruct`,
  `sentence-transformers/all-MiniLM-L6-v2`, `facebook/bart-large-mnli`
* embedding `text-embedding-3-small`
* `llm_endpoint` for the local Ollama base URL
* secret references for `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` and
  `HUGGINGFACE_API_KEY`

That is the expected shape for a Go target: dependency and literal detection
plus env resolution, with prompts, agents and tools added by the agentic pass.

## 5. Demo narrative

1. Show `internal/config/catalog.go` — the model registry, including the three
   Hugging Face models.
2. Run the scan and open the report.
3. Contrast with the Python sample: Go yields fewer Tier-1 findings because the
   AST analysis is shallower, which is a good teaching point about scanner
   maturity per language.
4. Show the prompt templates as first-class AI assets; they are embedded in the
   binary but still visible to source scanning.
5. Show the LangChainGo dependency and the escalation chain as an example of
   framework-level detection, and the HF zero-shot scorer as a second model
   family in the same service.

## 6. Container variant

```bash
docker build -t triage-agent:demo .
cisco-aibom analyze triage-agent:demo -o json -O aibom-image.json \
  --llm-model gpt-4o-mini
```

Scanning the image exercises the container path: the scanner extracts the Go
build context and reports the same models and prompts with image digest
provenance attached.
