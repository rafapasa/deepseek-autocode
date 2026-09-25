package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafapasa/deepseek-autocode/internal/config"
	"github.com/rafapasa/deepseek-autocode/internal/core"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type Orchestrator struct {
	client core.LlmInterface
}

func NewOrchestrator(llm LlmInterface) *Orchestrator {
	return &Orchestrator{client: llm}
}

func (o *Orchestrator) Start(req dto.Request) error {
	fmt.Println("[AutoCode] Iniciando")
	fmt.Printf("[AutoCode] Raiz: %s\n", req.Raiz)
	fmt.Printf("[AutoCode] Demanda: %s\n\n", req.Demanda)

	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(req)

	messages := []dto.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	tools := core.BuildTools()
	executor := &core.ToolExecutor{Raiz: req.Raiz}

	maxIterations := 60
	totalTokens := 0

	for i := 0; i < maxIterations; i++ {
		fmt.Printf("\n[AutoCode] --- Turno %d ---\n", i+1)

		resp, err := o.client.Chat(messages, tools)
		if err != nil {
			return fmt.Errorf("erro no chat: %v", err)
		}

		choice := resp.Choices[0]
		totalTokens += resp.Usage.TotalTokens
		fmt.Printf("[DeepSeek] tokens: %d (acumulado: %d) | finish: %s\n",
			resp.Usage.TotalTokens, totalTokens, choice.FinishReason)

		if choice.FinishReason == "length" {
			fmt.Println("[AutoCode] AVISO: output truncado (max_tokens). Reduza o tamanho das escritas.")
		}

		msg := choice.Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) == 0 {
			fmt.Printf("\n[DeepSeek] Resposta final: %s\n", msg.Content)
			break
		}

		for _, call := range msg.ToolCalls {
			fmt.Printf("[Tool] %s(%s)\n", call.Function.Name, truncate(call.Function.Arguments, 120))

			if call.Function.Name == "done" {
				var p struct {
					Summary string `json:"summary"`
				}
				json.Unmarshal([]byte(call.Function.Arguments), &p)
				fmt.Printf("\n[AutoCode] CONCLUÍDO: %s\n", p.Summary)
				fmt.Printf("[AutoCode] Tokens totais: %d\n", totalTokens)
				return nil
			}

			result, err := executor.Execute(call)
			if err != nil {
				result = fmt.Sprintf("ERRO: %v", err)
				fmt.Printf("[Tool] erro: %v\n", err)
			} else {
				fmt.Printf("[Tool] ok (%d bytes)\n", len(result))
			}

			messages = append(messages, dto.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	fmt.Printf("\n[AutoCode] Atingiu maxIterations (%d). Tokens: %d\n", maxIterations, totalTokens)
	return nil
}

// ============================================================
// PROMPTS
// ============================================================

func buildSystemPrompt() string {
	return `Você é um engenheiro de software sênior executando uma tarefa cirúrgica em um projeto Go.

Você tem tools disponíveis:
- read_file: lê o conteúdo de um arquivo
- write_file: cria ou sobrescreve um arquivo
- list_dir: lista diretório (use só se absolutamente necessário)
- run_command: NÃO USE. Não é sua função.
- done: sinaliza conclusão

REGRAS CRÍTICAS:
- Leia APENAS os arquivos listados em "ARQUIVOS PERMITIDOS". Nada mais.
- NUNCA leia arquivos fora da lista. Se a tarefa menciona um arquivo, ele está na lista.
- NUNCA rode comandos (go build, go test, go fmt). Não é sua função.
- NUNCA explore diretórios por curiosidade. Vá direto ao ponto.
- NUNCA re-leia um arquivo que você já leu nesta sessão.
- Execute as tarefas na ordem apresentada (id 1, 2, 3, ...).
- Para cada tarefa, edite o arquivo indicado seguindo a descrição ao pé da letra.
- Se a descrição tem blocos de código, copie-os fielmente (assinaturas, WHERE, nomes).
- Se a descrição diz "REMOVER: X, Y", remova exatamente X e Y.
- Se a descrição diz "NÃO TOCAR: Z", não toque em Z.
- Quando todas as tarefas terminarem, chame done com um resumo do que foi feito.
- Se encontrar ambiguidade, escolha a interpretação mais literal da descrição.`
}

func buildUserPrompt(req dto.Request) string {
	var b strings.Builder

	b.WriteString("DEMANDA:\n")
	b.WriteString(req.Demanda)
	b.WriteString("\n\n")

	b.WriteString("RAIZ DO PROJETO:\n")
	b.WriteString(req.Raiz)
	b.WriteString("\n\n")

	if req.Estrutura != nil {
		estruturaStr, _ := json.MarshalIndent(req.Estrutura, "", "  ")
		b.WriteString("ESTRUTURA DO PROJETO:\n")
		b.Write(estruturaStr)
		b.WriteString("\n\n")
	}

	b.WriteString("ARQUIVOS PERMITIDOS (só estes podem ser lidos e editados):\n")
	for _, a := range req.Arquivos {
		b.WriteString("- ")
		b.WriteString(a)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("TAREFAS (execute na ordem):\n\n")
	for _, t := range req.Tarefas {
		fmt.Fprintf(&b, "========================================\n")
		fmt.Fprintf(&b, "TAREFA %s — [%s] %s\n", t.ID, t.Tipo, t.Arquivo)
		fmt.Fprintf(&b, "========================================\n")
		b.WriteString(t.Descricao)
		b.WriteString("\n\n")
	}

	b.WriteString("REGRAS:\n")
	for i, r := range req.Rules {
		fmt.Fprintf(&b, "%d. %s\n", i+1, r)
	}
	b.WriteString("\n")
	b.WriteString("Comece lendo os arquivos listados em ordem. Não explore outros. Não rode comandos. Execute as tarefas e chame done ao terminar.")

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
