.PHONY: build test run tidy lint docker clean

BINARY := bin/triage-agent

build:
	go build -o $(BINARY) ./cmd/triage-agent

test:
	go test ./...

run:
	AI_PROVIDER=$${AI_PROVIDER:-stub} go run ./cmd/triage-agent

tidy:
	go mod tidy

lint:
	go vet ./...

docker:
	docker build -t triage-agent:demo .

aibom:
	cisco-aibom analyze . -o json -O aibom-report.json \
		--llm-model $${AIBOM_LLM_MODEL:-gpt-4o-mini}

clean:
	rm -rf bin aibom-report.json aibom-report.html aibom.cdx.json
