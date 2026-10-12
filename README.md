# deepseek-autocode
Geraador de codigo pela DeepSek atreavés de API


cd /home/opc/prj/deepseek-autocode

# UI (background)
make ui

# UI (foreground, pra ver logs ao vivo)
make ui-fg

# Parar UI
make kill

# Reiniciar
make restart

# CLI
make cli ISSUE=/home/opc/prj/issues/mcp-server-openerp/issue-cliente.json

# Trocar porta
make ui PORT=9090

# Ajuda
make help

# Limpar
make clean

# Senha da UI (obrigatória)
# A UI não sobe sem senha. Gere o hash (digite a senha) e coloque no .env:
go run ./cmd/hashpw
#   DSAC_UI_PASSWORD_HASH='$2a$10$...'   (preferido, bcrypt)
#   DSAC_UI_PASSWORD=...                 (alternativa em texto puro)
#   DSAC_UI_AUTH=off                     (desativa; só para dev local)
# Sessão: cookie de 12h, logout em /logout, 5 tentativas erradas/min por IP.

# run_command (tool do LLM)
#   DSAC_SHELL_ROOT=/home/opc/prj   DSAC_SHELL_TIMEOUT=120
#   DSAC_SHELL_MODE=allowlist|denylist   DSAC_SHELL_ALLOW=bin1,bin2
