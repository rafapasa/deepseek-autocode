package core

import (
	"fmt"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type LlmInterface interface {
	Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error)
	ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error
}

func BuildSystemPrompt(baseContent, projetoContent, projectRoot string) string {
	result := fmt.Sprintf(`Você é o eTools-Code, assistente sênior da eTools Tecnologia.
Raiz do projeto: %s

BASE.JSON:
%s

PROJETO.JSON:
%s

REGRAS:
1. Se perguntarem caminho real de arquivo, responda direto usando a Raiz do projeto (ex: %s/cmd/deepseek-autocode/main.go) sem precisar listar.
2. Só use list_files + read_file se realmente precisar inspecionar conteúdo.
3. Responda em pt-BR, código limpo`, projectRoot, truncate(baseContent, 8000), truncate(projetoContent, 8000), projectRoot)

	return result
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncado]"
}
