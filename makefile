# ============================================================
# deepseek-autocode — Makefile
# ============================================================

BIN_DIR    := bin
BIN_NAME   := deepseek-autocode
BIN_PATH   := $(BIN_DIR)/$(BIN_NAME)
CMD_PATH   := ./cmd/deepseek-autocode
SERVICE    := dsac
PORT       := 8080

# ------------------------------------------------------------
# Default
# ------------------------------------------------------------
.PHONY: help
help:
	@echo "Comandos disponíveis:"
	@echo "  make build     — compila o binário"
	@echo "  make restart   — mata, compila, corrige SELinux e reinicia"
	@echo "  make stop      — para o serviço e mata processos órfãos"
	@echo "  make start     — inicia o serviço"
	@echo "  make status    — mostra status do serviço"
	@echo "  make logs      — segue os logs em tempo real"
	@echo "  make clean     — remove binário e cache"
	@echo "  make reset     — reset-failed do systemd e reinicia"

# ------------------------------------------------------------
# Build
# ------------------------------------------------------------
.PHONY: build
build:
	@echo "🔨 Compilando $(BIN_NAME)..."
	@go build -o $(BIN_PATH) $(CMD_PATH)
	@echo "🔐 Corrigindo contexto SELinux..."
	@sudo restorecon -v $(BIN_PATH) 2>/dev/null || true
	@echo "✅ Binário pronto: $(BIN_PATH)"

# ------------------------------------------------------------
# Kill (mata processos órfãos + libera a porta)
# ------------------------------------------------------------
.PHONY: kill
kill:
	@echo "🛑 Parando serviço..."
	@-sudo systemctl stop $(SERVICE) 2>/dev/null || true
	@echo "🛑 Matando processos órfãos do $(BIN_NAME)..."
	@-pkill -f "$(BIN_NAME)" 2>/dev/null || true
	@echo "🔍 Verificando porta $(PORT)..."
	@-sudo fuser -k $(PORT)/tcp 2>/dev/null || true
	@-sudo ss -tlnp | grep :$(PORT) || echo "   ✅ Porta $(PORT) livre"
	@echo "🧹 Resetando contador de falhas do systemd..."
	@-sudo systemctl reset-failed $(SERVICE) 2>/dev/null || true

# ------------------------------------------------------------
# Stop
# ------------------------------------------------------------
.PHONY: stop
stop:
	@sudo systemctl stop $(SERVICE)
	@-pkill -f "$(BIN_NAME)" 2>/dev/null || true
	@-sudo fuser -k $(PORT)/tcp 2>/dev/null || true
	@echo "✅ Parado."

# ------------------------------------------------------------
# Start
# ------------------------------------------------------------
.PHONY: start
start:
	@sudo systemctl start $(SERVICE)
	@sleep 1
	@sudo systemctl status $(SERVICE) --no-pager | head -12

# ------------------------------------------------------------
# Restart (kill → build → start)
# ------------------------------------------------------------
.PHONY: restart
restart: kill build
	@echo "🚀 Iniciando serviço..."
	@sudo systemctl start $(SERVICE)
	@sleep 1
	@sudo systemctl status $(SERVICE) --no-pager | head -12
	@echo ""
	@echo "✅ Pronto. Acesse: https://dsac.etoolstec.com.br"
	@echo "📋 Logs: make logs"

# ------------------------------------------------------------
# Reset (limpa tudo do systemd e reinicia)
# ------------------------------------------------------------
.PHONY: reset
reset:
	@sudo systemctl stop $(SERVICE) 2>/dev/null || true
	@sudo systemctl reset-failed $(SERVICE) 2>/dev/null || true
	@sudo systemctl start $(SERVICE)
	@sleep 1
	@sudo systemctl status $(SERVICE) --no-pager | head -12

# ------------------------------------------------------------
# Logs / Status
# ------------------------------------------------------------
.PHONY: logs
logs:
	@sudo journalctl -u $(SERVICE) -f

.PHONY: status
status:
	@sudo systemctl status $(SERVICE) --no-pager
	@echo ""
	@echo "🔍 Porta $(PORT):"
	@-sudo ss -tlnp | grep :$(PORT) || echo "   ❌ Nada escutando"

# ------------------------------------------------------------
# Clean
# ------------------------------------------------------------
.PHONY: clean
clean:
	@echo "🧹 Limpando..."
	@rm -f $(BIN_PATH)
	@go clean -cache -testcache 2>/dev/null || true
	@echo "✅ Limpo."

# ------------------------------------------------------------
# Dev: roda sem systemd (para debug)
# ------------------------------------------------------------
.PHONY: dev
dev: build
	@echo "🚧 Rodando em foreground (Ctrl+C para sair)..."
	@$(BIN_PATH) --ui --port $(PORT)
