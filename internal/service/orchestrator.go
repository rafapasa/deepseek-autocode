package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rafapasa/deepseek-autocode/internal/core"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type Request struct {
	Demanda   string        `json:"demanda"`
	Estrutura []interface{} `json:"estrutura"`
	Raiz      string        `json:"raiz"`
	Rules     []string      `json:"rules"`
}

type Orchestrator struct {
	client core.Llminterface
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		client: core.NewDeepSeekClient(),
	}
}

// remove os delimitadores de código (```json, ```, etc) e texto extra
func cleanJSONResponse(raw string) string {
	// Remove delimitadores ```json ... ``` ou ``` ... ```
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	matches := re.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Se não encontrar, tenta extrair apenas o JSON
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

func (o *Orchestrator) Start(req Request) error {
	fmt.Println("[AutoCode] Demanda solicitada")

	rulesText := ""
	for i, rule := range req.Rules {
		rulesText += fmt.Sprintf("%d. %s\n", i+1, rule)
	}

	estruturaStr, _ := json.MarshalIndent(req.Estrutura, "", "  ")

	prompt := fmt.Sprintf(`Demanda: %s

Estrutura do projeto:
%s

Regras de codigo:
%s

Responda APENAS com JSON no formato:
{
  "status": "need_files",
  "files_needed": [
    {"path": "caminho/arquivo.dart", "description": "descrição"}
  ]
}
`, req.Demanda, string(estruturaStr), rulesText)

	resp, err := o.client.Chat(prompt)
	if err != nil {
		return err
	}

	// Limpa a resposta antes de parsear
	cleanResp := cleanJSONResponse(resp)

	var needed dto.FilesNeededResponse
	if err := json.Unmarshal([]byte(cleanResp), &needed); err != nil {
		return fmt.Errorf("erro ao parsear JSON: %v\nResposta limpa: %s", err, cleanResp)
	}

	if needed.Status != "need_files" {
		return fmt.Errorf("status inesperado: %s", needed.Status)
	}

	var files []dto.FileContent
	for _, f := range needed.FilesNeeded {
		fmt.Printf("[DeepSeek] Solicita arquivo %s\n", f.Path)

		fullPath := filepath.Join(req.Raiz, f.Path)
		if !core.FileExists(fullPath) {
			fmt.Printf("[AutoCode] Arquivo solicitado não encontrado: %s\n", fullPath)
		} else {
			content, err := core.ReadFile(fullPath)
			if err != nil {
				return fmt.Errorf("erro ao ler %s: %v", fullPath, err)
			}

			files = append(files, dto.FileContent{
				Path:    f.Path,
				Content: content,
			})
			fmt.Printf("[AutoCode] Arquivo %s enviado\n", f.Path)
		}

	}

	payload, _ := json.Marshal(files)

	prompt2 := fmt.Sprintf(`Aqui estão os arquivos que você pediu:

%s

Agora gere os arquivos modificados/criados e retorne APENAS JSON no formato:
{
  "status": "done",
  "files": [
    {"path": "caminho/arquivo.dart", "content": "// codigo"}
  ]
}

Regras de codigo:
%s
`, string(payload), rulesText)

	resp2, err := o.client.Chat(prompt2)
	if err != nil {
		return err
	}

	// Limpa a resposta antes de parsear
	cleanResp2 := cleanJSONResponse(resp2)

	var final dto.FinalResponse
	if err := json.Unmarshal([]byte(cleanResp2), &final); err != nil {
		return fmt.Errorf("erro ao parsear JSON final: %v\nResposta limpa: %s", err, cleanResp2)
	}

	for _, f := range final.Files {
		fullPath := filepath.Join(req.Raiz, f.Path)

		if core.FileExists(fullPath) {
			backup := fullPath + ".bak"
			if err := os.Rename(fullPath, backup); err != nil {
				return fmt.Errorf("erro ao criar backup de %s: %v", f.Path, err)
			}
			fmt.Printf("[AutoCode] Backup criado: %s.bak\n", f.Path)
		}

		if err := core.WriteFile(fullPath, f.Content); err != nil {
			return fmt.Errorf("erro ao salvar %s: %v", f.Path, err)
		}

		fmt.Printf("[AutoCode] Arquivo salvo: %s\n", f.Path)
	}

	return nil
}
