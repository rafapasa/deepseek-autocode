package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

func GetTools() []dto.Tool {
	return []dto.Tool{
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "list_files", Description: "Lista arquivos de um diretório do projeto",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"dir": map[string]interface{}{"type": "string"}}, "required": []string{"dir"}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "read_file", Description: "Lê conteúdo de um arquivo",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}}, "required": []string{"path"}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "write_file", Description: "Cria ou sobrescreve arquivo no projeto",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}}, "required": []string{"path", "content"}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "apply_patch", Description: "Aplica patch diff em arquivo existente",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "diff": map[string]interface{}{"type": "string"}}, "required": []string{"path", "diff"}},
			},
		},
	}
}

type ToolResult struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

func ExecuteTool(projectRoot, toolName, argsJSON string) ToolResult {
	var args map[string]string
	json.Unmarshal([]byte(argsJSON), &args)
	if len(args) == 0 {
		var generic map[string]interface{}
		json.Unmarshal([]byte(argsJSON), &generic)
		args = make(map[string]string)
		for k, v := range generic {
			if s, ok := v.(string); ok {
				args[k] = s
			} else {
				b, _ := json.Marshal(v)
				args[k] = string(b)
			}
		}
	}

	safeJoin := func(p string) (string, error) {
		clean := filepath.Clean(p)
		if strings.Contains(clean, "..") {
			return "", fmt.Errorf("path fora do projeto")
		}
		return filepath.Join(projectRoot, clean), nil
	}

	switch toolName {
	case "list_files":
		dir := args["dir"]
		if dir == "" {
			dir = "."
		}
		full, err := safeJoin(dir)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		var out []string
		for _, e := range entries {
			prefix := "FILE"
			if e.IsDir() {
				prefix = "DIR "
			}
			out = append(out, fmt.Sprintf("%s %s", prefix, e.Name()))
		}
		return ToolResult{Success: true, Content: strings.Join(out, "\n")}
	case "read_file":
		full, err := safeJoin(args["path"])
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		return ToolResult{Success: true, Content: string(data)}
	case "write_file":
		full, err := safeJoin(args["path"])
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		os.MkdirAll(filepath.Dir(full), 0755)
		if err := os.WriteFile(full, []byte(args["content"]), 0644); err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		return ToolResult{Success: true, Content: fmt.Sprintf("Arquivo escrito: %s (%d bytes)", args["path"], len(args["content"]))}
	default:
		return ToolResult{Success: false, Error: "tool desconhecida: " + toolName}
	}
}
