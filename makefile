# ──────────────────────────────────────────────────────────────
#  CodeGen Makefile
#  Targets: build  run  stop  clean  help
# ──────────────────────────────────────────────────────────────

BINARY    := bin/code2json
GO_MAIN   := ./cmd/main.go
TEMPLATES := templates
PORT_GO   := 8080
PORT_HTML := 3000
PID_GO    := .pid_go
PID_HTML  := .pid_html

.PHONY: all build run stop clean help

all: run

## build: compile the Go binary
build:
	@echo "  ▶ Compiling Go application..."
	@mkdir -p bin
	@go build -o $(BINARY) $(GO_MAIN)
	@echo "  ✔ Binary ready → $(BINARY)"

## run: build + start Go API + start HTML server
run: build
	@echo "  ▶ Starting Go API on http://localhost:$(PORT_GO) ..."
	@./$(BINARY) & echo $$! > $(PID_GO)
	@sleep 0.5
	@echo "  ▶ Starting HTML server on http://localhost:$(PORT_HTML) ..."
	@cd $(TEMPLATES) && python3 -m http.server $(PORT_HTML) --bind 127.0.0.1 > /dev/null 2>&1 & echo $$! > ../$(PID_HTML)
	@sleep 0.3
	@echo ""
	@echo "  ┌────────────────────────────────────────────────────┐"
	@echo "  │  🚀  CodeGen is running                             │"
	@echo "  │                                                     │"
	@echo "  │  UI  →  http://localhost:$(PORT_HTML)                   │"
	@echo "  │  API →  http://localhost:$(PORT_GO)/api/generate       │"
	@echo "  │                                                     │"
	@echo "  │  Run  make stop  to shut everything down            │"
	@echo "  └────────────────────────────────────────────────────┘"

## stop: kill both servers
stop:
	@echo "  ▶ Stopping servers..."
	@[ -f $(PID_GO) ]   && kill $$(cat $(PID_GO))   2>/dev/null || true; rm -f $(PID_GO)
	@[ -f $(PID_HTML) ] && kill $$(cat $(PID_HTML)) 2>/dev/null || true; rm -f $(PID_HTML)
	@echo "  ✔ Done."

## clean: stop + remove build artefacts
clean: stop
	@rm -rf bin $(PID_GO) $(PID_HTML)
	@echo "  ✔ Cleaned."

## help: show available targets
help:
	@echo ""
	@echo "  CodeGen — available make targets:"
	@grep -E '^## ' Makefile | sed 's/## /    /'
	@echo ""