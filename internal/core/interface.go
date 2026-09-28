package core

import (
	"fmt"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

// LlmInterface é a fonte única. Mantive a assinatura original do seu Chat() para não quebrar.
type LlmInterface interface {
	Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error)
	ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error
}

func BuildSystemPrompt(baseContent, projetoContent, projectRoot string) string {
	return fmt.Sprintf(`Você é o eTools-Code, assistente sênior da eTools Tecnologia.
Raiz real do projeto: %s
Cores: Azul #1E3A5F e Verde #16A34A - use em qualquer UI gerada.

BASE.JSON:
%s

PROJETO.JSON:
%s

REGRAS:
1. Se perguntarem caminho real de arquivo, responda direto usando a Raiz real (ex: %s/cmd/deepseek-autocode/main.go) sem precisar listar diretórios.
2. Quando precisar corrigir teste ou implementar funcionalidade em arquivo específico, siga fluxo seguro: read_file do alvo, read_file de relacionados se precisar, write_file com correção, done.
3. Use list_dir apenas quando precisar confirmar estrutura, não por curiosidade.
4. Responda em pt-BR, código limpo, sem expor caminhos fora da raiz.
`, projectRoot, truncate(baseContent, 8000), truncate(projetoContent, 8000), projectRoot)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncado]"
}
