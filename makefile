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
	@sudo cp -f $(BIN_PATH) /usr/local/bin/$(BIN_NAME)
	@sudo chmod 755 /usr/local/bin/$(BIN_NAME)
	@echo "🔐 Corrigindo contexto SELinux..."
	@sudo restorecon -v $(BIN_PATH) /usr/local/bin/$(BIN_NAME) 2>/dev/null || true
	@echo "✅ Binário pronto: $(BIN_PATH) e /usr/local/bin/$(BIN_NAME)"

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


# ------------------------------------------------------------
# Diagnóstico do unit (sandbox / read-only)
# ------------------------------------------------------------
.PHONY: unit
unit:
	@echo "=== systemctl cat ==="
	@sudo systemctl cat $(SERVICE) || true
	@echo ""
	@echo "=== show Protect* ==="
	@sudo systemctl show $(SERVICE) -p User,Group,ProtectSystem,ProtectHome,ReadOnlyPaths,ReadWritePaths,BindPaths,TemporaryFileSystem,NoNewPrivileges,PrivateTmp,RootDirectory,RootImage,ProtectProc || true
	@echo ""
	@echo "=== binary ==="
	@ls -la /usr/local/bin/$(BIN_NAME) $(BIN_PATH) 2>/dev/null || true
	@echo ""
	@echo "=== mount /home ==="
	@findmnt /home /home/opc /home/opc/prj 2>/dev/null || true

# ------------------------------------------------------------
# Libera escrita nos projetos (ProtectHome=read-only no unit)
# ------------------------------------------------------------
.PHONY: unlock-fs
unlock-fs:
	@echo "🔓 Ajustando ReadWritePaths do $(SERVICE)..."
	@sudo mkdir -p /etc/systemd/system/$(SERVICE).service.d
	@printf '%s\n' '[Service]' 'ProtectHome=read-only' 'ReadWritePaths=/home/opc/prj /home/opc/.ds-ac' | sudo tee /etc/systemd/system/$(SERVICE).service.d/write.conf >/dev/null
	@sudo systemctl daemon-reload
	@sudo systemctl restart $(SERVICE)
	@sleep 1
	@sudo systemctl show $(SERVICE) -p ReadWritePaths,ProtectHome
	@echo "✅ Serviço reiniciado com escrita em /home/opc/prj"

# ------------------------------------------------------------
# Commit e push para origin/main
# ------------------------------------------------------------
.PHONY: git-push-main
git-push-main:
	@git add -A
	@git status
	@git -c user.name="Rafael Alberto Pasa" -c user.email="rafapasa@gmail.com" commit -m "feat(ui): chat multiplo, explorer de fontes, logs e layout dsac" || true
	@git push origin main
	@git log -1 --oneline
	@git status -sb
