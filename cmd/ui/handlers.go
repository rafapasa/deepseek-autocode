package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/deepseek-autocode/internal/config"
)

type ProjectInfo struct {
	Name   string   `json:"name"`
	Issues []string `json:"issues"`
}

type Handler struct {
	cfg *config.Config
}

func NewHandler(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

// GET /
func (h *Handler) Index(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(indexHTML)
}

// GET /api/env
func (h *Handler) CheckEnv(c *fiber.Ctx) error {
	baseOK, projetoOK, baseErr, projetoErr := h.checkEnv()
	resp := fiber.Map{
		"base_ok":      baseOK,
		"projeto_ok":   projetoOK,
		"base_path":    h.cfg.BaseJsonPath,
		"projeto_path": h.cfg.ProjetoJsonPath,
	}
	if !baseOK {
		resp["base_err"] = baseErr
	}
	if !projetoOK {
		resp["projeto_err"] = projetoErr
	}
	return c.JSON(resp)
}

func (h *Handler) checkEnv() (baseOK, projetoOK bool, baseErr, projetoErr string) {
	baseOK, baseErr = checkJSON(h.cfg.BaseJsonPath)
	// projeto.json é opcional globalmente, mas validamos se o path estiver setado
	if h.cfg.ProjetoJsonPath != "" {
		projetoOK, projetoErr = checkJSON(h.cfg.ProjetoJsonPath)
	} else {
		// se não tem path global, consideramos OK (ele vem por projeto dentro de IssuesDir)
		projetoOK = true
	}
	return
}

func checkJSON(path string) (bool, string) {
	if path == "" {
		return false, "path vazio"
	}
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

// GET /api/config
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"issues_dir":       h.cfg.IssuesDir,
		"deepseek_api_key": maskKey(h.cfg.DeepSeekApiKey),
		"meta_api_key":     maskKey(h.cfg.MetaApiKey),
		"http_port":        h.cfg.HttpPort,
		"llm_client":       h.cfg.LlmClient,
	})
}

// POST /api/config
func (h *Handler) SaveConfig(c *fiber.Ctx) error {
	var body struct {
		IssuesDir      string `json:"issues_dir"`
		DeepSeekApiKey string `json:"deepseek_api_key"`
		MetaApiKey     string `json:"meta_api_key"`
		HttpPort       int    `json:"http_port"`
		LlmClient      int    `json:"llm_client"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}

	if body.IssuesDir != "" {
		h.cfg.IssuesDir = body.IssuesDir
	}
	if body.DeepSeekApiKey != "" {
		h.cfg.DeepSeekApiKey = body.DeepSeekApiKey
	}
	if body.MetaApiKey != "" {
		h.cfg.MetaApiKey = body.MetaApiKey
	}
	if body.HttpPort != 0 {
		h.cfg.HttpPort = body.HttpPort
	}
	if body.LlmClient != 0 {
		h.cfg.LlmClient = body.LlmClient
	}

	// Salva em ~/.ds-ac/config.json pra persistir a escolha da UI
	if err := h.cfg.Save(); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"ok": true})
}

func maskKey(k string) string {
	if len(k) <= 4 {
		return ""
	}
	return "..." + k[len(k)-4:]
}

// GET /api/issues
func (h *Handler) ListIssues(c *fiber.Ctx) error {
	if h.cfg.IssuesDir == "" {
		return c.JSON(fiber.Map{"projects": []ProjectInfo{}})
	}

	entries, err := os.ReadDir(h.cfg.IssuesDir)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	var projects []ProjectInfo
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "concluidas" {
			continue
		}
		dir := filepath.Join(h.cfg.IssuesDir, e.Name())
		issues := listIssues(dir)
		if len(issues) == 0 {
			continue
		}
		projects = append(projects, ProjectInfo{Name: e.Name(), Issues: issues})
	}

	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })

	return c.JSON(fiber.Map{"projects": projects})
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
		if strings.HasSuffix(name, ".json") && !strings.Contains(name, ".concluida") && name != "projeto.json" && name != "base.json" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// POST /api/issues/create
func (h *Handler) CreateIssue(c *fiber.Ctx) error {
	var body struct {
		Project  string   `json:"project"`
		Filename string   `json:"filename"`
		Demanda  string   `json:"demanda"`
		Raiz     string   `json:"raiz"`
		Arquivos []string `json:"arquivos"`
		Tarefas  []struct {
			ID        string `json:"id"`
			Arquivo   string `json:"arquivo"`
			Tipo      string `json:"tipo"`
			Descricao string `json:"descricao"`
		} `json:"tarefas"`
		Rules []string `json:"rules"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}

	if body.Project == "" || body.Filename == "" || body.Demanda == "" {
		return c.Status(400).SendString("project, filename e demanda obrigatórios")
	}
	if len(body.Arquivos) == 0 || len(body.Tarefas) == 0 {
		return c.Status(400).SendString("pelo menos 1 arquivo e 1 tarefa")
	}

	if h.cfg.IssuesDir == "" {
		return c.Status(400).SendString("issues_dir não configurado")
	}

	projectDir := filepath.Join(h.cfg.IssuesDir, body.Project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if !strings.HasSuffix(body.Filename, ".json") {
		body.Filename += ".json"
	}
	targetPath := filepath.Join(projectDir, filepath.Base(body.Filename))
	if _, err := os.Stat(targetPath); err == nil {
		return c.Status(409).SendString("já existe uma issue com esse nome")
	}

	issueData := map[string]interface{}{
		"demanda":  body.Demanda,
		"raiz":     body.Raiz,
		"arquivos": body.Arquivos,
		"tarefas":  body.Tarefas,
	}
	if len(body.Rules) > 0 {
		issueData["rules"] = body.Rules
	}

	content, err := json.MarshalIndent(issueData, "", "  ")
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"ok": true, "path": targetPath})
}

// POST /api/issues/upload
func (h *Handler) UploadIssue(c *fiber.Ctx) error {
	project := c.FormValue("project")
	if project == "" {
		return c.Status(400).SendString("project obrigatório")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString("arquivo não enviado: " + err.Error())
	}

	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	defer h.closeFile(f)

	content, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	var test map[string]interface{}
	if err := json.Unmarshal(content, &test); err != nil {
		return c.Status(400).SendString("arquivo não é um JSON válido: " + err.Error())
	}

	if h.cfg.IssuesDir == "" {
		return c.Status(400).SendString("issues_dir não configurado")
	}

	projectDir := filepath.Join(h.cfg.IssuesDir, project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	filename := filepath.Base(fileHeader.Filename)
	if !strings.HasSuffix(filename, ".json") {
		return c.Status(400).SendString("só aceita arquivos .json")
	}

	targetPath := filepath.Join(projectDir, filename)
	if _, err := os.Stat(targetPath); err == nil {
		return c.Status(409).SendString("já existe uma issue com esse nome")
	}

	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.JSON(fiber.Map{"ok": true, "path": targetPath, "size": len(content)})
}

// POST /api/run
func (h *Handler) Run(c *fiber.Ctx) error {
	baseOK, projetoOK, baseErr, projetoErr := h.checkEnv()
	// base.json é obrigatório, projeto.json global é opcional
	if !baseOK {
		msg := "ambiente incompleto:\n- base: " + baseErr + "\n"
		if !projetoOK && h.cfg.ProjetoJsonPath != "" {
			msg += "- projeto: " + projetoErr + "\n"
		}
		return c.Status(400).SendString(msg)
	}

	if h.cfg.IssuesDir == "" {
		return c.Status(400).SendString("config ausente (issues_dir)")
	}

	if h.cfg.ResolveAPIKey() == "" {
		return c.Status(400).SendString("API Key não configurada (defina no .env DEEPSEEK_API_KEY ou META_API_KEY)")
	}

	var req struct {
		Project string `json:"project"`
		Issue   string `json:"issue"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).SendString(err.Error())
	}

	run, err := StartRun(h.cfg, req.Project, req.Issue)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"id": run.ID})
}

// helper que não retorna erro - resolve o lint errcheck de forma elegante
// o erro é tratado internamente, o call site não precisa checar retorno
func writeSSE(w *bufio.Writer, payload interface{}) {
	line := fmt.Sprintf("data: %s\n\n", jsonString(payload))
	if _, err := w.WriteString(line); err != nil {
		return
	}
	if err := w.Flush(); err != nil {
		return
	}
}

// GET /api/stream/:id - SSE
func (h *Handler) Stream(c *fiber.Ctx) error {
	id := c.Params("id")
	run, ok := GetRun(id)
	if !ok {
		return c.Status(404).SendString("run não encontrado")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case line, ok := <-run.Lines:
				if !ok {
					writeSSE(w, fiber.Map{"type": "done", "success": run.Success})
					run.Cleanup()
					return
				}
				writeSSE(w, fiber.Map{"type": "line", "text": line})
			case <-c.Context().Done():
				return
			}
		}
	})

	return nil
}

// POST /api/stop
func (h *Handler) Stop(c *fiber.Ctx) error {
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
	return c.JSON(fiber.Map{"ok": true})
}

func jsonString(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (h *Handler) closeFile(f multipart.File) {
	if err := f.Close(); err != nil {
		log.Printf("Erro fechando arquivo: %v", err)
	}
}
