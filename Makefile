.PHONY: build clean test

BINARY_NAME=stackobrite-agent
VERSION=v0.1.0
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-s -w -X 'github.com/stackobrite/stackobrite-agent/internal/handler.Version=$(VERSION)' -X 'github.com/stackobrite/stackobrite-agent/internal/handler.BuildDate=$(BUILD_DATE)' -X 'github.com/stackobrite/stackobrite-agent/internal/handler.GitCommit=$(GIT_COMMIT)'

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/agent

build-local:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/agent

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).bak $(BINARY_NAME).new

test:
	go test ./...

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	install -m 644 deploy/stackobrite-agent.service /etc/systemd/system/
	systemctl daemon-reload
	systemctl enable stackobrite-agent
	systemctl start stackobrite-agent

dev: build-local
	AGENT_TOKEN=dev-token-12345 ./$(BINARY_NAME)
