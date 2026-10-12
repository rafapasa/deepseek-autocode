package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

// RunCommandTool é a definição compartilhada da tool run_command (issue e chat).
func RunCommandTool() dto.Tool {
	return dto.Tool{
		Type: "function",
		Function: dto.ToolFunction{
			Name:        "run_command",
			Description: "Executa um comando permitido (go, gofmt, make, dart, flutter, git status/diff/log/show/branch, ls, cat, grep, rg, tree, head, tail, wc, diff) dentro do projeto. Sem pipes, redirecionamentos ou encadeamento. Retorna exit code e saída (máx 64KB). Use para build, testes e formatação quando a demanda pedir.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Comando a executar (ex: 'go build ./...', 'go test ./...', 'make build')",
					},
					"cwd": map[string]interface{}{
						"type":        "string",
						"description": "Opcional. Diretório relativo à raiz do projeto (padrão '.')",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func BuildTools() []dto.Tool {
	return []dto.Tool{
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "read_file",
				Description: "Lê o conteúdo de um arquivo. Use APENAS para arquivos que você vai modificar ou que servem de referência direta (máximo 2-3 por chamada).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Caminho relativo à raiz",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "write_file",
				Description: "Cria ou sobrescreve um arquivo com conteúdo completo. Use SOMENTE quando a demanda pedir modificação.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Caminho relativo à raiz",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Conteúdo completo do arquivo",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "list_dir",
				Description: "Lista arquivos e pastas de um diretório. Use APENAS se a estrutura fornecida na issue não bater com a realidade. Não use para 'explorar'.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Caminho relativo (use '.' para a raiz)",
						},
					},
					"required": []string{"path"},
				},
			},
		},
		RunCommandTool(),
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "done",
				Description: "Sinaliza conclusão. Sempre chame ao final. O summary deve ser conciso (2-5 linhas) e é o resumo que o humano vai ler.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"summary": map[string]interface{}{
							"type":        "string",
							"description": "Resumo do que foi feito",
						},
					},
					"required": []string{"summary"},
				},
			},
		},
	}
}

type ToolExecutor struct {
	Raiz string
}

func (t *ToolExecutor) Execute(call dto.ToolCall) (string, error) {
	switch call.Function.Name {
	case "read_file":
		return t.readFile(call.Function.Arguments)
	case "write_file":
		return t.writeFile(call.Function.Arguments)
	case "list_dir":
		return t.listDir(call.Function.Arguments)
	case "run_command":
		return t.runCommand(call.Function.Arguments)
	case "done":
		return "", nil
	default:
		return "", fmt.Errorf("tool desconhecida: %s", call.Function.Name)
	}
}

// safePath: confina à raiz da issue (lexical) e à raiz global, resolvendo symlinks.
func (t *ToolExecutor) safePath(rel string) (string, error) {
	joined, err := SafeJoin(t.Raiz, rel)
	if err != nil {
		return "", err
	}
	return ResolveSafePath(t.Raiz, joined)
}

func (t *ToolExecutor) readFile(args string) (string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}
	full, err := t.safePath(p.Path)
	if err != nil {
		return "", err
	}
	content, err := ReadFile(full)
	if err != nil {
		return "", err
	}
	// Trunca em 64KB para evitar estouro de tokens/contexto
	const maxBytes = 64 * 1024
	if len(content) > maxBytes {
		return content[:maxBytes] + fmt.Sprintf("\n\n... [truncado: %d de %d bytes]", maxBytes, len(content)), nil
	}
	return content, nil
}

func (t *ToolExecutor) writeFile(args string) (string, error) {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}
	full, err := t.safePath(p.Path)
	if err != nil {
		return "", err
	}
	if err := WriteFile(full, p.Content); err != nil {
		return "", err
	}
	return fmt.Sprintf("arquivo escrito: %s (%d bytes)", p.Path, len(p.Content)), nil
}

func (t *ToolExecutor) listDir(args string) (string, error) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}
	full, err := t.safePath(p.Path)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(full)
	if err != nil {
		return "", fmt.Errorf("erro ao ler diretório: %v", err)
	}

	var sb strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		typeChar := "-"
		if entry.IsDir() {
			typeChar = "d"
		}
		fmt.Fprintf(&sb, "%s %10d %s\n", typeChar, info.Size(), entry.Name())
	}
	return sb.String(), nil
}

func (t *ToolExecutor) runCommand(args string) (string, error) {
	var p struct {
		Command string `json:"command"`
		Cwd     string `json:"cwd"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}
	return RunShellCommand(t.Raiz, p.Cwd, p.Command)
}
