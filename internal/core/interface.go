package core

import (
	"fmt"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type LlmInterface interface {
	Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error)
	ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error
}

func BuildSystemPrompt(baseContent, projetoContent string) string {
	return fmt.Sprintf(`Você é o eTools-Code, assistente sênior da eTools Tecnologia.
Cores: Azul #1E3A5F e Verde #16A34A - use em qualquer UI gerada.

BASE.JSON:
%s

PROJETO.JSON:
%s

REGRAS:
1. Use list_files + read_file antes de afirmar
2. Crie com write_file, sempre dentro do projeto
3. Responda em pt-BR, código limpo
`, truncate(baseContent, 8000), truncate(projetoContent, 8000))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncado]"
}
