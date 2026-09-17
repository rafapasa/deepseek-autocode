package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rafapasa/deepseek-autocode/cmd/ui"
	"github.com/rafapasa/deepseek-autocode/internal/config"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
	"github.com/rafapasa/deepseek-autocode/internal/service"
)

const (
	defaultBasePath = "/home/opc/prj/issues/base.json"
)

func main() {
	uiMode := flag.Bool("ui", false, "modo interface web")
	port := flag.String("port", "8080", "porta do servidor web (modo ui)")
	apikey := flag.String("key", "", "chave da api da DeepSeek (opcional — fallback: config.json, env)")
	flag.Parse()

	// Resolve a chave em ordem: flag > env > config.json
	key := *apikey
	if key == "" {
		key = os.Getenv("DEEPSEEK_API_KEY")
	}
	if key == "" {
		if c, err := ui.LoadConfig(); err == nil && c.DeepSeekApiKey != "" {
			key = c.DeepSeekApiKey
		}
	}
	if key == "" {
		fmt.Println("❌ API Key da DeepSeek não encontrada.")
		fmt.Println("   Defina com: --key <chave>")
		fmt.Println("   Ou:  export DEEPSEEK_API_KEY=<chave>")
		fmt.Println("   Ou:  configure no modal ⚙ config da UI")
		os.Exit(1)
	}

	// Propaga pro env (subprocessos herdam)
	os.Setenv("DEEPSEEK_API_KEY", key)

	cfg := config.NewConfig(key)

	if *uiMode {
		if err := ui.Start(*port); err != nil {
			fmt.Printf("❌ Erro no servidor: %v\n", err)
			os.Exit(1)
		}
		return
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Uso:")
		fmt.Println("  deepseek-autocode <issue.json>")
		fmt.Println("  deepseek-autocode --ui [--port 8080]")
		os.Exit(1)
	}

	tarefaPath := args[0]
	basePath := envOr("DS_AC_BASE", defaultBasePath)
	projetoPath := filepath.Join(filepath.Dir(tarefaPath), "projeto.json")

	base, err := loadJSON(basePath)
	if err != nil {
		fmt.Printf("❌ base.json inválido (%s): %v\n", basePath, err)
		os.Exit(1)
	}
	projeto, err := loadJSON(projetoPath)
	if err != nil {
		fmt.Printf("❌ projeto.json inválido (%s): %v\n", projetoPath, err)
		os.Exit(1)
	}
	tarefa, err := loadJSON(tarefaPath)
	if err != nil {
		fmt.Printf("❌ tarefa inválida (%s): %v\n", tarefaPath, err)
		os.Exit(1)
	}

	req, err := merge(base, projeto, tarefa)
	if err != nil {
		fmt.Printf("❌ merge: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[ds-ac] raiz: %s\n", req.Raiz)
	fmt.Printf("[ds-ac] arquivos: %d | tarefas: %d | rules: %d\n\n",
		len(req.Arquivos), len(req.Tarefas), len(req.Rules))

	o := service.NewOrchestrator(cfg)
	if err := o.Start(req); err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		os.Exit(1)
	}
}

// ============================================================
// CARREGAMENTO / MERGE
// ============================================================

func loadJSON(path string) (map[string]interface{}, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("JSON inválido: %w", err)
	}
	return out, nil
}

func merge(base, projeto, tarefa map[string]interface{}) (dto.Request, error) {
	var out dto.Request

	out.Rules = appendStringSlice(
		toStringSlice(base["rules"]),
		toStringSlice(projeto["rules"]),
		toStringSlice(tarefa["rules"]),
	)

	out.Raiz = firstNonEmpty(
		toString(tarefa["raiz"]),
		toString(projeto["raiz"]),
		toString(base["raiz"]),
	)

	switch {
	case tarefa["estrutura"] != nil:
		out.Estrutura = tarefa["estrutura"]
	case projeto["estrutura"] != nil:
		out.Estrutura = projeto["estrutura"]
	case base["estrutura"] != nil:
		out.Estrutura = base["estrutura"]
	}

	out.Demanda = toString(tarefa["demanda"])
	out.Arquivos = toStringSlice(tarefa["arquivos"])

	if rawTarefas, ok := tarefa["tarefas"].([]interface{}); ok {
		for _, item := range rawTarefas {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			out.Tarefas = append(out.Tarefas, dto.Tarefa{
				ID:        toString(m["id"]),
				Arquivo:   toString(m["arquivo"]),
				Tipo:      toString(m["tipo"]),
				Descricao: toString(m["descricao"]),
			})
		}
	}

	if out.Raiz == "" {
		return out, fmt.Errorf("raiz ausente (nem tarefa, nem projeto, nem base definiram)")
	}
	if out.Demanda == "" {
		return out, fmt.Errorf("demanda ausente na tarefa")
	}
	if len(out.Arquivos) == 0 {
		return out, fmt.Errorf("arquivos ausente na tarefa")
	}
	if len(out.Tarefas) == 0 {
		return out, fmt.Errorf("tarefas ausente na tarefa")
	}

	return out, nil
}

// ============================================================
// HELPERS
// ============================================================

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func appendStringSlice(slices ...[]string) []string {
	var total int
	for _, s := range slices {
		total += len(s)
	}
	out := make([]string, 0, total)
	for _, s := range slices {
		out = append(out, s...)
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
