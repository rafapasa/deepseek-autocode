package ui

import (
	"encoding/json"
	"fmt"
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
		json.NewEncoder(w).Encode(c)
	case "POST":
		var c Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := SaveConfig(&c); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", 405)
	}
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

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	// Valida base + projeto antes de qualquer coisa
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
