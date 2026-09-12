package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/rafapasa/deepseek-autocode/internal/core"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type Orchestrator struct {
	client core.LlmInterface
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		client: core.NewDeepSeekClient(),
	}
}

const (
	maxIterationsPerPart = 15
)

// cleanJSONResponse remove ```json ... ``` e texto extra
func cleanJSONResponse(raw string) string {
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	matches := re.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	start := strings.Index(raw, "{")
	if start == -1 {
		return raw
	}
	end := strings.LastIndex(raw, "}")
	if end == -1 {
		return raw
	}
	return raw[start : end+1]
}

// ===== PLANEJAMENTO =====

// ===== PLANEJAMENTO (loop: need_files → lê → repete até direct/split) =====

func (o *Orchestrator) plan(req dto.Request) (*dto.PlanResponse, error) {
	rulesText := ""
	for i, r := range req.Rules {
		rulesText += fmt.Sprintf("%d. %s\n", i+1, r)
	}

	estruturaStr, _ := json.MarshalIndent(req.Estrutura, "", "  ")

	systemPrompt := `Você é um engenheiro de software que analisa demandas e decide como executá-las.

Você tem 3 status possíveis:

1. "need_files": você precisa ler alguns arquivos para decidir. Liste os arquivos no formato:
   {"status": "need_files", "files_needed": [{"path": "...", "reason": "..."}]}
   Leia APENAS os arquivos essenciais. Não peça 20 arquivos.

2. "direct": a demanda é pequena (até 3 arquivos) e você já tem tudo. Vá direto para execução.

3. "split": a demanda é grande (7+ arquivos, ou múltiplos módulos independentes). Divida em partes:
   {"status": "split", "parts": [
     {"id": 1, "demanda": "...", "files": ["path/a.go", "path/b.go"]},
     {"id": 2, "demanda": "...", "files": ["path/c.go"]}
   ]}
   Cada parte deve ter 2-3 arquivos no máximo e ser executável em sequência.

Responda APENAS com JSON, nada mais.`

	userPrompt := fmt.Sprintf(`Demanda:
%s

Estrutura do projeto:
%s

Regras:
%s

Analise e responda com o status apropriado.`,
		req.Demanda, string(estruturaStr), rulesText)

	messages := []dto.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	for iter := 0; iter < 5; iter++ {
		resp, err := o.client.Chat(messages, nil)
		if err != nil {
			return nil, fmt.Errorf("erro no planejamento: %v", err)
		}

		content := resp.Choices[0].Message.Content
		clean := cleanJSONResponse(content)

		var plan dto.PlanResponse
		if err := json.Unmarshal([]byte(clean), &plan); err != nil {
			return nil, fmt.Errorf("erro ao parsear plano (iter %d): %v\n%s", iter, err, clean)
		}

		fmt.Printf("[ds-ac] [planejamento iter %d] status: %s\n", iter+1, plan.Status)

		switch plan.Status {
		case "direct", "split":
			return &plan, nil

		case "need_files":
			// precisa ler arquivos
			fmt.Printf("[ds-ac] Planejador pediu arquivos: ")

			// A resposta precisa ter os arquivos. Como o PlanResponse não tem esse campo,
			// vamos parsear direto do JSON cru.
			var raw struct {
				FilesNeeded []struct {
					Path   string `json:"path"`
					Reason string `json:"reason"`
				} `json:"files_needed"`
			}
			if err := json.Unmarshal([]byte(clean), &raw); err != nil {
				return nil, fmt.Errorf("erro ao parsear files_needed: %v", err)
			}

			if len(raw.FilesNeeded) == 0 {
				return nil, fmt.Errorf("status need_files sem files_needed")
			}

			// imprime lista
			var paths []string
			for _, f := range raw.FilesNeeded {
				paths = append(paths, f.Path)
				fmt.Printf("%s ", f.Path)
			}
			fmt.Println()

			// lê os arquivos
			var filesContent strings.Builder
			for _, f := range raw.FilesNeeded {
				full, err := core.SafeJoin(req.Raiz, f.Path)
				if err != nil {
					filesContent.WriteString(fmt.Sprintf("\n===== %s =====\n[ERRO: %v]\n", f.Path, err))
					continue
				}
				content, err := core.ReadFile(full)
				if err != nil {
					filesContent.WriteString(fmt.Sprintf("\n===== %s =====\n[ERRO: %v]\n", f.Path, err))
					continue
				}
				filesContent.WriteString(fmt.Sprintf("\n===== %s =====\n%s\n", f.Path, content))
			}

			// adiciona a resposta da LLM e o conteúdo dos arquivos no histórico
			messages = append(messages, dto.Message{Role: "assistant", Content: content})
			messages = append(messages, dto.Message{
				Role:    "user",
				Content: fmt.Sprintf("Aqui estão os arquivos:\n%s\n\nAgora decida: direct ou split. Responda APENAS com JSON.", filesContent.String()),
			})

		default:
			return nil, fmt.Errorf("status desconhecido: %s", plan.Status)
		}
	}

	return nil, fmt.Errorf("planejamento não convergiu em 5 iterações")
}

// ===== EXECUÇÃO DE UMA PARTE =====

func (o *Orchestrator) executePart(req dto.Request, part dto.Part, previousSummary string) (string, error) {
	rulesText := ""
	for i, r := range req.Rules {
		rulesText += fmt.Sprintf("%d. %s\n", i+1, r)
	}

	// lê os arquivos da parte
	var filesContent strings.Builder
	for _, path := range part.Files {
		full, err := core.SafeJoin(req.Raiz, path)
		if err != nil {
			return "", err
		}
		content, err := core.ReadFile(full)
		if err != nil {
			filesContent.WriteString(fmt.Sprintf("\n===== %s =====\n[ERRO: %v]\n", path, err))
			continue
		}
		filesContent.WriteString(fmt.Sprintf("\n===== %s =====\n%s\n", path, content))
	}

	summaryContext := ""
	if previousSummary != "" {
		summaryContext = fmt.Sprintf("\nResumo das partes anteriores:\n%s\n", previousSummary)
	}

	systemPrompt := `Você executa UMA tarefa pequena. Não é agente.

REGRAS:
1. Faça EXATAMENTE o que a demanda da parte pede. Nada a mais.
2. Não explore. Não builde. Não teste. Não rode git.
3. Se precisar rodar algo (a demanda pediu explicitamente), use run_command.
4. Modifique APENAS os arquivos envolvidos.
5. Quando terminar, chame done com um resumo curto (2-5 linhas).`

	userPrompt := fmt.Sprintf(`Demanda da parte:
%s

Arquivos envolvidos:
%s
%s
Regras gerais:
%s

Execute a demanda. Use write_file para cada arquivo modificado. No final chame done com o resumo.`,
		part.Demanda, filesContent.String(), summaryContext, rulesText)

	messages := []dto.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	tools := core.BuildTools()
	executor := &core.ToolExecutor{Raiz: req.Raiz}

	for i := 0; i < maxIterationsPerPart; i++ {
		fmt.Printf("    [turno %d] ", i+1)

		resp, err := o.client.Chat(messages, tools)
		if err != nil {
			return "", fmt.Errorf("erro no chat: %v", err)
		}

		choice := resp.Choices[0]
		msg := choice.Message

		if len(msg.ToolCalls) == 0 {
			fmt.Printf("texto final (%d tokens)\n", resp.Usage.TotalTokens)
			if msg.Content != "" {
				fmt.Printf("    [llm] %s\n", truncate(msg.Content, 200))
			}
			break
		}

		fmt.Printf("%d tool(s)\n", len(msg.ToolCalls))
		messages = append(messages, msg)

		for _, call := range msg.ToolCalls {
			if call.Function.Name == "done" {
				var p struct {
					Summary string `json:"summary"`
				}
				json.Unmarshal([]byte(call.Function.Arguments), &p)
				fmt.Printf("    [done] %s\n", truncate(p.Summary, 200))
				return p.Summary, nil
			}

			fmt.Printf("    [tool] %s(%s)\n", call.Function.Name, truncate(call.Function.Arguments, 100))
			result, err := executor.Execute(call)
			if err != nil {
				result = fmt.Sprintf("ERRO: %v", err)
			}

			messages = append(messages, dto.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	return "", fmt.Errorf("parte não chamou done em %d turnos", maxIterationsPerPart)
}

// ===== START =====

func (o *Orchestrator) Start(req dto.Request) error {
	fmt.Println("========================================")
	fmt.Println("[ds-ac] Iniciando")
	fmt.Printf("[ds-ac] Raiz: %s\n", req.Raiz)
	fmt.Printf("[ds-ac] Demanda: %s\n", req.Demanda)
	fmt.Println("========================================")

	// ===== 1. Planejamento =====
	fmt.Println("\n[ds-ac] Planejando...")
	plan, err := o.plan(req)
	if err != nil {
		return fmt.Errorf("erro no planejamento: %v", err)
	}

	fmt.Printf("[ds-ac] Status do plano: %s\n", plan.Status)

	switch plan.Status {
	case "direct":
		// executa como uma única parte, com os arquivos que a LLM descobrir
		fmt.Println("[ds-ac] Demanda pequena, executando direto.")

		// pede os arquivos via tool calling normal
		part := dto.Part{
			ID:      1,
			Demanda: req.Demanda,
			Files:   extractFilesFromEstrutura(req.Estrutura), // ou vazio se não tiver
		}
		if len(part.Files) == 0 {
			// sem lista explícita: deixa a LLM usar list_dir/read_file
			fmt.Println("[ds-ac] Sem lista de arquivos, deixando a LLM explorar.")
		}
		summary, err := o.executePart(req, part, "")
		if err != nil {
			return err
		}
		fmt.Printf("\n========================================\n")
		fmt.Printf("[ds-ac] CONCLUÍDO\n")
		fmt.Printf("[ds-ac] Resumo: %s\n", summary)
		fmt.Printf("========================================\n")
		return nil

	case "split":
		fmt.Printf("[ds-ac] Dividido em %d partes.\n", len(plan.Parts))
		for i, p := range plan.Parts {
			fmt.Printf("    parte %d: %s (%d arquivos)\n", p.ID, truncate(p.Demanda, 80), len(p.Files))
			_ = i
		}

		var allSummaries []string
		previousSummary := ""

		for _, part := range plan.Parts {
			fmt.Printf("\n========================================\n")
			fmt.Printf("[ds-ac] EXECUTANDO PARTE %d/%d\n", part.ID, len(plan.Parts))
			fmt.Printf("[ds-ac] Demanda: %s\n", part.Demanda)
			fmt.Printf("[ds-ac] Arquivos: %v\n", part.Files)
			fmt.Println("========================================")

			summary, err := o.executePart(req, part, previousSummary)
			if err != nil {
				fmt.Printf("\n[ds-ac] ERRO na parte %d: %v\n", part.ID, err)
				fmt.Printf("[ds-ac] Parando. Resumo das partes concluídas:\n")
				for _, s := range allSummaries {
					fmt.Printf("  - %s\n", s)
				}
				return err
			}

			fmt.Printf("[ds-ac] Parte %d concluída: %s\n", part.ID, summary)

			allSummaries = append(allSummaries, fmt.Sprintf("Parte %d: %s", part.ID, summary))
			previousSummary = strings.Join(allSummaries, "\n")
		}

		fmt.Printf("\n========================================\n")
		fmt.Printf("[ds-ac] CONCLUÍDO - %d partes executadas\n", len(plan.Parts))
		fmt.Printf("[ds-ac] Resumo consolidado:\n")
		for _, s := range allSummaries {
			fmt.Printf("  %s\n", s)
		}
		fmt.Printf("========================================\n")
		return nil

	case "need_files":
		return fmt.Errorf("status need_files ainda não implementado no planejamento. Adicione os arquivos na issue diretamente.")

	default:
		return fmt.Errorf("status desconhecido: %s", plan.Status)
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// extractFilesFromEstrutura é um placeholder - a estrutura tem dirs/files aninhados.
// Retorna lista vazia (a LLM usa list_dir se precisar).
func extractFilesFromEstrutura(estrutura []interface{}) []string {
	return nil
}
