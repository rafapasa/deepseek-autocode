package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/deepseek-autocode/internal/config"
)

type ProjectInfo struct {
	Name          string   `json:"name"`
	Issues        []string `json:"issues"`
	ProjetoExists bool     `json:"projeto_exists"`
}

type Handler struct{ cfg *config.Config }

func NewHandler(cfg *config.Config) *Handler { return &Handler{cfg: cfg} }

func (h *Handler) Index(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(indexHTML)
}

func (h *Handler) CheckEnv(c *fiber.Ctx) error {
	baseOK, projetoOK, baseErr, projetoErr := h.checkEnv()
	resp := fiber.Map{"base_ok": baseOK, "projeto_ok": projetoOK, "base_path": h.cfg.BaseJsonPath, "projeto_path": h.cfg.ProjetoJsonPath, "issues_dir": h.cfg.IssuesDir}
	if !baseOK {
		resp["base_err"] = baseErr
	}
	if !projetoOK {
		resp["projeto_err"] = projetoErr
	}
	return c.JSON(resp)
}

func (h *Handler) checkEnv() (bool, bool, string, string) {
	baseOK, baseErr := checkJSON(h.cfg.BaseJsonPath)
	projetoOK := true
	var projetoErr string
	if h.cfg.ProjetoJsonPath != "" {
		projetoOK, projetoErr = checkJSON(h.cfg.ProjetoJsonPath)
	}
	return baseOK, projetoOK, baseErr, projetoErr
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
		return false, "JSON inválido: " + err.Error()
	}
	return true, ""
}

func (h *Handler) GetConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"issues_dir": h.cfg.IssuesDir, "http_port": h.cfg.HttpPort, "deepseek_api_key": maskKey(h.cfg.DeepSeekApiKey), "meta_api_key": maskKey(h.cfg.MetaApiKey), "llm_client": h.cfg.LlmClient})
}

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

// CONCLUIDAS IGNORADA TOTALMENTE
func (h *Handler) ListIssues(c *fiber.Ctx) error {
	if h.cfg.IssuesDir == "" {
		return c.JSON(fiber.Map{"projects": []ProjectInfo{}, "base_exists": false})
	}
	entries, err := os.ReadDir(h.cfg.IssuesDir)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	var projects []ProjectInfo
	baseExists := false
	for _, e := range entries {
		if !e.IsDir() {
			if e.Name() == "base.json" {
				baseExists = true
			}
			continue
		}
		if strings.EqualFold(e.Name(), "concluidas") {
			continue
		}
		dir := filepath.Join(h.cfg.IssuesDir, e.Name())
		issues := listIssues(dir)
		projetoExists := fileExists(filepath.Join(dir, "projeto.json"))
		projects = append(projects, ProjectInfo{Name: e.Name(), Issues: issues, ProjetoExists: projetoExists})
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Name < projects[j].Name })
	return c.JSON(fiber.Map{"projects": projects, "base_exists": baseExists})
}

func (h *Handler) ListProjects(c *fiber.Ctx) error {
	if h.cfg.IssuesDir == "" {
		return c.JSON([]string{})
	}
	entries, err := os.ReadDir(h.cfg.IssuesDir)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && !strings.EqualFold(e.Name(), "concluidas") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return c.JSON(out)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func listIssues(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			if strings.EqualFold(e.Name(), "concluidas") {
				continue
			}
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

func (h *Handler) GetBase(c *fiber.Ctx) error {
	path := h.cfg.BaseJsonPath
	if path == "" {
		path = filepath.Join(h.cfg.IssuesDir, "base.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"content": string(data), "path": path})
}

func (h *Handler) SaveBase(c *fiber.Ctx) error {
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	var test map[string]interface{}
	if err := json.Unmarshal([]byte(body.Content), &test); err != nil {
		return c.Status(400).SendString("JSON inválido: " + err.Error())
	}
	path := h.cfg.BaseJsonPath
	if path == "" {
		path = filepath.Join(h.cfg.IssuesDir, "base.json")
	}
	if err := os.WriteFile(path, []byte(body.Content), 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *Handler) UploadBase(c *fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	defer h.closeFile(f)

	content, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	path := h.cfg.BaseJsonPath
	if path == "" {
		path = filepath.Join(h.cfg.IssuesDir, "base.json")
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *Handler) GetProjeto(c *fiber.Ctx) error {
	project := c.Params("project")
	path := filepath.Join(h.cfg.IssuesDir, project, "projeto.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"content": string(data), "path": path})
}

func (h *Handler) SaveProjeto(c *fiber.Ctx) error {
	project := c.Params("project")
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	path := filepath.Join(h.cfg.IssuesDir, project, "projeto.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if err := os.WriteFile(path, []byte(body.Content), 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *Handler) UploadProjeto(c *fiber.Ctx) error {
	project := c.Params("project")
	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	defer h.closeFile(f)

	content, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	path := filepath.Join(h.cfg.IssuesDir, project, "projeto.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *Handler) GetIssueFile(c *fiber.Ctx) error {
	project := c.Params("project")
	file := c.Params("*")
	if file == "" {
		file = c.Params("file")
	}
	if strings.Contains(file, "..") || strings.Contains(strings.ToLower(file), "concluidas") {
		return c.Status(400).SendString("concluidas ignorada")
	}
	path := filepath.Join(h.cfg.IssuesDir, project, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"content": string(data), "path": path})
}

func (h *Handler) SaveIssueFile(c *fiber.Ctx) error {
	project := c.Params("project")
	file := c.Params("*")
	if file == "" {
		file = c.Params("file")
	}
	if strings.Contains(strings.ToLower(file), "concluidas") {
		return c.Status(400).SendString("concluidas ignorada")
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	path := filepath.Join(h.cfg.IssuesDir, project, file)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if err := os.WriteFile(path, []byte(body.Content), 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *Handler) UploadIssueReplace(c *fiber.Ctx) error {
	project := c.Params("project")
	file := c.Params("*")
	if file == "" {
		file = c.Params("file")
	}
	if strings.Contains(strings.ToLower(file), "concluidas") {
		return c.Status(400).SendString("concluidas ignorada")
	}
	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	defer h.closeFile(f)

	content, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	path := filepath.Join(h.cfg.IssuesDir, project, file)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func (h *Handler) CreateIssue(c *fiber.Ctx) error {
	var body struct {
		Project  string   `json:"project"`
		Filename string   `json:"filename"`
		Demanda  string   `json:"demanda"`
		Raiz     string   `json:"raiz"`
		Arquivos []string `json:"arquivos"`
		Tarefas  []struct {
			Id        string `json:"id"`
			Arquivo   string `json:"arquivo"`
			Tipo      string `json:"tipo"`
			Descricao string `json:"descricao"`
		} `json:"tarefas"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	projectDir := filepath.Join(h.cfg.IssuesDir, body.Project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if !strings.HasSuffix(body.Filename, ".json") {
		body.Filename += ".json"
	}
	targetPath := filepath.Join(projectDir, filepath.Base(body.Filename))
	content, err := json.MarshalIndent(map[string]interface{}{"demanda": body.Demanda, "raiz": body.Raiz, "arquivos": body.Arquivos, "tarefas": body.Tarefas}, "", " ")
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": targetPath})
}

func (h *Handler) UploadIssue(c *fiber.Ctx) error {
	project := c.FormValue("project")
	fh, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	defer h.closeFile(f)

	content, err := io.ReadAll(f)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	projectDir := filepath.Join(h.cfg.IssuesDir, project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	targetPath := filepath.Join(projectDir, filepath.Base(fh.Filename))
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": targetPath})
}

func (h *Handler) Run(c *fiber.Ctx) error {
	var req struct {
		Project string `json:"project"`
		Issue   string `json:"issue"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if strings.Contains(strings.ToLower(req.Issue), "concluidas") {
		return c.Status(400).SendString("concluidas ignorada")
	}
	run, err := StartRun(h.cfg, req.Project, req.Issue)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"id": run.ID})
}

func writeSSE(w *bufio.Writer, payload interface{}) {
	line := fmt.Sprintf("data: %s\n\n", jsonString(payload))
	_, _ = w.WriteString(line)
	_ = w.Flush()
}

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

func jsonString(v interface{}) string { b, _ := json.Marshal(v); return string(b) }

func (h *Handler) closeFile(f multipart.File) {
	if f != nil {
		_ = f.Close()
	}
}
