package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	LLM_DEEPSEEK = 1
	LLM_META     = 2
	LLM_GEMINI   = 3
)

type Config struct {
	// Fixa - vem do .env
	LlmClient      int    `json:"llm_client"`
	DeepSeekApiKey string `json:"deepseek_api_key"`
	MetaApiKey     string `json:"meta_api_key"`
	GeminiApiKey   string `json:"gemini_api_key"`
	HttpPort       int    `json:"http_port"`

	// Dinâmica - vem da UI e é salva em ~/.ds-ac/config.json, mas também pode vir do .env
	IssuesDir       string `json:"issues_dir"`
	BaseJsonPath    string `json:"base_path"`
	ProjetoJsonPath string `json:"projeto_path"`

	// interno
	configFilePath string `json:"-"`
}

// uiPersist é o que fica em ~/.ds-ac/config.json pra compatibilidade com a UI antiga
type uiPersist struct {
	IssuesDir      string `json:"issues_dir"`
	DeepSeekApiKey string `json:"deepseek_api_key,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "config.json")
}

func NewConfig() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		LlmClient:       getEnvAsInt("LLM_CLIENT", LLM_DEEPSEEK),
		DeepSeekApiKey:  getEnv("DEEPSEEK_API_KEY", ""),
		MetaApiKey:      getEnv("META_API_KEY", ""),
		GeminiApiKey:    getEnv("GEMINI_API_KEY", ""),
		HttpPort:        getEnvAsInt("HTTP_PORT", 8080),
		IssuesDir:       getEnv("ISSUES_DIR", "/home/opc/prj/issues"),
		BaseJsonPath:    getEnv("DS_AC_BASE", "/home/opc/prj/issues/base.json"),
		ProjetoJsonPath: getEnv("DS_AC_PROJETO", ""),
		configFilePath:  configPath(),
	}

	// Fallback: se ~/.ds-ac/config.json existir, usa pra preencher o que não veio do .env
	// Isso mantém sua UI funcionando quando o usuário salva pelo modal ⚙
	if data, err := os.ReadFile(cfg.configFilePath); err == nil {
		var persisted uiPersist
		if json.Unmarshal(data, &persisted) == nil {
			if cfg.IssuesDir == "/home/opc/prj/issues" && persisted.IssuesDir != "" {
				// só sobrescreve se o .env não definiu um custom
				if getEnv("ISSUES_DIR", "") == "" {
					cfg.IssuesDir = persisted.IssuesDir
				}
			}
			if cfg.DeepSeekApiKey == "" && persisted.DeepSeekApiKey != "" {
				cfg.DeepSeekApiKey = persisted.DeepSeekApiKey
			}
		}
	}

	return cfg
}

func (c *Config) ResolveAPIKey() string {
	switch c.LlmClient {
	case LLM_META:
		if c.MetaApiKey != "" {
			return c.MetaApiKey
		}
		return c.DeepSeekApiKey
	default:
		if c.DeepSeekApiKey != "" {
			return c.DeepSeekApiKey
		}
		return c.MetaApiKey
	}
}

func (c *Config) Save() error {
	// Salva em ~/.ds-ac/config.json pra UI continuar funcionando
	if err := os.MkdirAll(filepath.Dir(c.configFilePath), 0755); err != nil {
		return err
	}
	persist := uiPersist{
		IssuesDir:      c.IssuesDir,
		DeepSeekApiKey: c.DeepSeekApiKey,
	}
	data, err := json.MarshalIndent(persist, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configFilePath, data, 0644)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			return iv
		}
	}
	return defaultValue
}
