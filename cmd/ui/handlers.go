package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultBasePath    = "/home/opc/prj/issues/base.json"
	defaultProjetoPath = "/home/opc/prj/issues/projeto.json"
)

type ProjectInfo struct {
	Name   string   `json:"name"`
	Issues []string `json:"issues"`
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// checkEnv valida se base.json e projeto.json existem e são JSON válidos.
func checkEnv() (baseOK, projetoOK bool, baseErr, projetoErr string) {
	basePath := envOr("DS_AC_BASE", defaultBasePath)
	projetoPath := envOr("DS_AC_PROJETO", defaultProjetoPath)

	baseOK, baseErr = checkJSON(basePath)
	projetoOK, projetoErr = checkJSON(projetoPath)
	return
}

func checkJSON(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "arquivo não encontrado: " + path
		}
		return false, "erro ao ler " + path + ": " + err.Error()
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return false, "JSON inválido em " + path + ": " + err.Error()
	}
	return true, ""
}

func handleEnv(w http.ResponseWriter, r *http.Request) {
	baseOK, projetoOK, baseErr, projetoErr := checkEnv()
	resp := map[string]interface{}{
		"base_ok":      baseOK,
		"projeto_ok":   projetoOK,
		"base_path":    envOr("DS_AC_BASE", defaultBasePath),
		"projeto_path": envOr("DS_AC_PROJETO", defaultProjetoPath),
	}
	if !baseOK {
		resp["base_err"] = baseErr
	}
	if !projetoOK {
		resp["projeto_err"] = projetoErr
	}
	json.NewEncoder(w).Encode(resp)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		c, err := LoadConfig()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// Não expõe a chave completa na resposta GET (só os últimos 4 dígitos)
		resp := map[string]interface{}{
			"issues_dir":       c.IssuesDir,
			"deepseek_api_key": maskKey(c.DeepSeekApiKey),
		}
		json.NewEncoder(w).Encode(resp)
	case "POST":
		// Carrega config atual e faz merge (pra não perder a key se o form não enviar)
		current, _ := LoadConfig()
		if current == nil {
			current = &Config{}
		}

		var body struct {
			IssuesDir      string `json:"issues_dir"`
			DeepSeekApiKey string `json:"deepseek_api_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		current.IssuesDir = body.IssuesDir
		// Se o campo veio preenchido, atualiza. Se veio vazio, mantém o que já tinha.
		if body.DeepSeekApiKey != "" {
			current.DeepSeekApiKey = body.DeepSeekApiKey
		}

		if err := SaveConfig(current); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

// maskKey mostra só os últimos 4 caracteres: "sk-...abcd"
func maskKey(k string) string {
	if len(k) <= 4 {
		return ""
	}
	return "..." + k[len(k)-4:]
}

func handleIssues(w http.ResponseWriter, r *http.Request) {
	c, err := LoadConfig()
	if err != nil || c.IssuesDir == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"projects": []ProjectInfo{}})
		return
	}

	entries, err := os.ReadDir(c.IssuesDir)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var projects []ProjectInfo
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "concluidas" {
			continue
		}
		dir := filepath.Join(c.IssuesDir, e.Name())
		issues := listIssues(dir)
		if len(issues) == 0 {
			continue
		}
		projects = append(projects, ProjectInfo{Name: e.Name(), Issues: issues})
	}

	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })

	json.NewEncoder(w).Encode(map[string]interface{}{"projects": projects})
}

func listIssues(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") && !strings.Contains(name, ".concluida") {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// ============================================================
// CRIAÇÃO DE ISSUE (formulário)
// ============================================================

func handleCreateIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var body struct {
		Project  string   `json:"project"`
		Filename string   `json:"filename"`
		Demanda  string   `json:"demanda"`
		Raiz     string   `json:"raiz,omitempty"`
		Arquivos []string `json:"arquivos"`
		Tarefas  []struct {
			ID        string `json:"id"`
			Arquivo   string `json:"arquivo"`
			Tipo      string `json:"tipo"`
			Descricao string `json:"descricao"`
		} `json:"tarefas"`
		Rules []string `json:"rules,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "JSON inválido: "+err.Error(), 400)
		return
	}

	// Validações básicas
	if body.Project == "" {
		http.Error(w, "project obrigatório", 400)
		return
	}
	if body.Filename == "" {
		http.Error(w, "filename obrigatório", 400)
		return
	}
	if !strings.HasSuffix(body.Filename, ".json") {
		body.Filename += ".json"
	}
	if body.Demanda == "" {
		http.Error(w, "demanda obrigatória", 400)
		return
	}
	if len(body.Arquivos) == 0 {
		http.Error(w, "pelo menos 1 arquivo é obrigatório", 400)
		return
	}
	if len(body.Tarefas) == 0 {
		http.Error(w, "pelo menos 1 tarefa é obrigatória", 400)
		return
	}

	// Sanitiza filename (sem path traversal)
	body.Filename = filepath.Base(body.Filename)
	if !strings.HasSuffix(body.Filename, ".json") {
		body.Filename += ".json"
	}

	c, err := LoadConfig()
	if err != nil || c.IssuesDir == "" {
		http.Error(w, "config ausente (issues_dir não configurado)", 400)
		return
	}

	projectDir := filepath.Join(c.IssuesDir, body.Project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		http.Error(w, "erro ao criar pasta do projeto: "+err.Error(), 500)
		return
	}

	// Monta o JSON no formato esperado pelo merge (tarefa)
	payload := map[string]interface{}{
		"demanda":  body.Demanda,
		"arquivos": body.Arquivos,
		"tarefas":  body.Tarefas,
	}
	if body.Raiz != "" {
		payload["raiz"] = body.Raiz
	}
	if len(body.Rules) > 0 {
		payload["rules"] = body.Rules
	}

	// Verifica se já existe
	targetPath := filepath.Join(projectDir, body.Filename)
	if _, err := os.Stat(targetPath); err == nil {
		http.Error(w, "já existe uma issue com esse nome", 409)
		return
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		http.Error(w, "erro ao serializar: "+err.Error(), 500)
		return
	}
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		http.Error(w, "erro ao salvar: "+err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"path": targetPath,
	})
}

// ============================================================
// UPLOAD DE ISSUE (multipart)
// ============================================================

func handleUploadIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	// Limite de 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "erro ao parsear form: "+err.Error(), 400)
		return
	}

	project := r.FormValue("project")
	if project == "" {
		http.Error(w, "project obrigatório", 400)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "arquivo não enviado: "+err.Error(), 400)
		return
	}
	defer file.Close()

	// Valida que é JSON
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "erro ao ler arquivo: "+err.Error(), 500)
		return
	}
	var test map[string]interface{}
	if err := json.Unmarshal(content, &test); err != nil {
		http.Error(w, "arquivo não é um JSON válido: "+err.Error(), 400)
		return
	}

	c, err := LoadConfig()
	if err != nil || c.IssuesDir == "" {
		http.Error(w, "config ausente (issues_dir não configurado)", 400)
		return
	}

	projectDir := filepath.Join(c.IssuesDir, project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		http.Error(w, "erro ao criar pasta: "+err.Error(), 500)
		return
	}

	// Sanitiza nome do arquivo
	filename := filepath.Base(header.Filename)
	if !strings.HasSuffix(filename, ".json") {
		http.Error(w, "só aceita arquivos .json", 400)
		return
	}

	targetPath := filepath.Join(projectDir, filename)
	if _, err := os.Stat(targetPath); err == nil {
		http.Error(w, "já existe uma issue com esse nome", 409)
		return
	}

	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		http.Error(w, "erro ao salvar: "+err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":   true,
		"path": targetPath,
		"size": len(content),
	})
}

// ============================================================
// EXECUÇÃO
// ============================================================

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	baseOK, projetoOK, baseErr, projetoErr := checkEnv()
	if !baseOK || !projetoOK {
		msg := "ambiente incompleto:\n"
		if !baseOK {
			msg += "- base: " + baseErr + "\n"
		}
		if !projetoOK {
			msg += "- projeto: " + projetoErr + "\n"
		}
		http.Error(w, msg, 400)
		return
	}

	c, err := LoadConfig()
	if err != nil || c.IssuesDir == "" {
		http.Error(w, "config ausente", 400)
		return
	}

	// Garante que a API Key está no env antes de rodar
	if os.Getenv("DEEPSEEK_API_KEY") == "" && c.DeepSeekApiKey != "" {
		os.Setenv("DEEPSEEK_API_KEY", c.DeepSeekApiKey)
	}

	var req struct {
		Project string `json:"project"`
		Issue   string `json:"issue"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	run, err := StartRun(c.IssuesDir, req.Project, req.Issue)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"id": run.ID})
}

func handleStream(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/stream/")
	run, ok := GetRun(id)
	if !ok {
		http.Error(w, "run não encontrado", 404)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	for {
		select {
		case line, ok := <-run.Lines:
			if !ok {
				fmt.Fprintf(w, "data: %s\n\n", jsonString(map[string]interface{}{"type": "done", "success": run.Success}))
				flusher.Flush()
				run.Cleanup()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonString(map[string]interface{}{"type": "line", "text": line}))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	var latest *Run
	runs.Range(func(k, v interface{}) bool {
		rr := v.(*Run)
		if latest == nil || rr.ID > latest.ID {
			latest = rr
		}
		return true
	})
	if latest != nil {
		latest.Stop()
	}
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
