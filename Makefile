.PHONY: install fmt fmt-check vet lint test test-race build tidy check clean

GOPATH_BIN := $(subst \,/,$(shell go env GOPATH))/bin
BINARY := scour
CMD := ./cmd/scour

ifeq ($(OS),Windows_NT)
	GOLANGCI := cmd /c "set GOTOOLCHAIN=local&& golangci-lint run ./..."
	BINARY_OUT := $(BINARY).exe
	RM := cmd /c del /Q /F
else
	GOLANGCI := GOTOOLCHAIN=local $(GOPATH_BIN)/golangci-lint run ./...
	BINARY_OUT := $(BINARY)
	RM := rm -f
endif

install:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/segmentio/golines@latest

fmt:
	$(GOPATH_BIN)/goimports -w .
	$(GOPATH_BIN)/golines -m 100 -w .

fmt-check:
	@out=`gofmt -s -l . | grep -v '^vendor/' || true`; \
	if [ -n "$$out" ]; then \
		echo "Files need formatting:"; echo "$$out"; exit 1; \
	fi

vet:
	@if [ -n "`find . -name '*.go' -not -path './vendor/*' -print -quit`" ]; then \
		go vet ./...; \
	else \
		echo "==> vet: no Go files yet, skipping"; \
	fi

lint:
	@if [ -n "`find . -name '*.go' -not -path './vendor/*' -print -quit`" ]; then \
		$(GOLANGCI); \
	else \
		echo "==> lint: no Go files yet, skipping"; \
	fi

test:
	@if [ -n "`find . -name '*.go' -not -path './vendor/*' -print -quit`" ]; then \
		go test -race -short ./...; \
	else \
		echo "==> test: no Go files yet, skipping"; \
	fi

build:
	@if [ -d "$(CMD)" ]; then \
		echo "==> build $(CMD)"; \
		go build -o $(BINARY_OUT) $(CMD); \
	else \
		echo "==> build ./... (no cmd/scour yet)"; \
		go build ./...; \
	fi

tidy:
	go mod tidy

check: fmt-check vet lint test

clean:
	$(RM) $(BINARY_OUT) coverage.out coverage.html
