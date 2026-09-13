BIN := ./bin/deepseek-autocode
MAIN := ./cmd/deepseek-autocode/main.go
PORT ?= 8080


.PHONY: all build cli ui kill clean help

all: build

## build: compila o binário
build:
	@echo ">> build"
	@go build -o $(BIN) .
	@echo "ok: $(BIN)"

## cli: roda no modo terminal. Uso: make cli ISSUE=caminho/issue.json
cli: build
	@if [ -z "$(ISSUE)" ]; then \
		echo "uso: make cli ISSUE=/caminho/issue.json"; \
		exit 1; \
	fi
	@$(BIN) $(ISSUE)

## ui: sobe a interface web em background
ui: build
	@pkill -f 'deepseek-autocode --ui' >/dev/null 2>&1 || true
	@nohup $(BIN) --ui --port $(PORT) > /tmp/dsac-ui.log 2>&1 & \
		sleep 1; \
		echo ">> ds-ac ui em http://localhost:$(PORT)"; \
		echo ">> log: tail -f /tmp/dsac-ui.log"

## ui-fg: sobe a interface web em foreground (pra debug)
ui-fg: build
	@$(BIN) --ui --port $(PORT)

## kill: mata o processo da UI
kill:
	@pkill -f 'deepseek-autocode --ui' && echo ">> ui parada" || echo ">> nada rodando"

## restart: reinicia a UI
restart: kill ui

## clean: remove binário e logs
clean:
	@rm -rf ./bin
	@rm -f /tmp/dsac-ui.log
	@echo ">> limpo"

## help: mostra esta ajuda
help:
	@echo "alvos disponíveis:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'