package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

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
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "run_command",
				Description: "Executa comando permitido (go, dart, flutter, make, ls, cat, find, grep). Use SOMENTE se a demanda pedir EXPLICITAMENTE para rodar algo (ex: 'rode go build', 'rode flutter test'). NUNCA use para 'verificar se funcionou'.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Comando a executar",
						},
					},
					"required": []string{"command"},
				},
			},
		},
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

func (t *ToolExecutor) safePath(rel string) (string, error) {
	return SafeJoin(t.Raiz, rel)
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
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", err
	}

	cmdStr := strings.TrimSpace(p.Command)
	if cmdStr == "" {
		return "", fmt.Errorf("comando vazio informado")
	}

	// Bloqueia encadeamento de comandos e operadores perigosos de shell
	forbiddenChars := []string{";", "&&", "||", "|", "`", "$", ">", "<", "\n"}
	for _, char := range forbiddenChars {
		if strings.Contains(cmdStr, char) {
			return "", fmt.Errorf("caractere de encadeamento não permitido no comando: %q", char)
		}
	}

	fields := strings.Fields(cmdStr)
	if len(fields) == 0 {
		return "", fmt.Errorf("comando inválido")
	}

	allowed := []string{"flutter", "dart", "go", "ls", "cat", "find", "grep", "make"}
	first := fields[0]
	ok := false
	for _, a := range allowed {
		if first == a {
			ok = true
			break
		}
	}
	if !ok {
		return "", fmt.Errorf("comando não permitido: %s", first)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	cmd.Dir = t.Raiz
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout (180s): %s", cmdStr)
	}
	if err != nil {
		return fmt.Sprintf("exit=%v\n%s", err, string(out)), nil
	}
	return string(out), nil
}
