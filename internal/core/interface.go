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
	if baseContent == "" {
		baseContent = "(base.json não encontrado)"
	}
	if projetoContent == "" {
		projetoContent = "(projeto.json não encontrado para este projeto)"
	}
	return fmt.Sprintf(`Você é o eTools-Code, assistente sênior da eTools Tecnologia.
Raiz real do código do projeto: %s

As seções BASE.JSON e PROJETO.JSON abaixo são obrigatórias. Use-as em toda decisão:
- regras, convenções, libs e padrões
- estrutura de pastas/arquivos do projeto
- nomes reais de pacotes, DTOs, services e testes
Não invente caminhos que contradigam o PROJETO.JSON. Se um arquivo listado lá existir, prefira lê-lo a explorar o disco sem necessidade.

BASE.JSON (regras globais, valem para todos os projetos):
%s

PROJETO.JSON (regras e estrutura deste projeto):
%s

REGRAS DE TRABALHO:
1. Responda em pt-BR. Código limpo. Não exponha caminhos fora da raiz.
2. Caminho real de arquivo: %s/<relativo>.
3. Para corrigir teste ou implementar: read_file do alvo e dos arquivos relacionados, depois write_file com o arquivo completo.
4. list_files só para confirmar estrutura, não por curiosidade.
5. Siga apperror, gokit/mapper, gokit/response e as rules do JSON acima.
6. Quando um teste panica por logger/zap nil, inicialize o logger no teste (ou injete nop logger) antes de chamar o código de produção.
`, projectRoot, truncate(baseContent, 12000), truncate(projetoContent, 28000), projectRoot)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncado]"
}
