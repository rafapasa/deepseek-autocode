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