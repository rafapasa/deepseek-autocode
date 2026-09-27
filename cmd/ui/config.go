package ui

// Este arquivo agora é só compatibilidade com o main.go antigo que chama ui.LoadConfig()
// A fonte da verdade agora é internal/config/config.go que lê do .env
// Mantemos esse arquivo pra não quebrar o `go build`, mas ele delega.

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rafapasa/deepseek-autocode/internal/config"
)

type Config struct {
	IssuesDir      string `json:"issues_dir"`
	DeepSeekApiKey string `json:"deepseek_api_key,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "config.json")
}

func LoadConfig() (*Config, error) {
	// Tenta ler o ~/.ds-ac/config.json legado
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fallback: pega do .env via config.NewConfig()
			c := config.NewConfig()
			return &Config{
				IssuesDir:      c.IssuesDir,
				DeepSeekApiKey: c.DeepSeekApiKey,
			}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func SaveConfig(c *Config) error {
	// Converte pro novo config e salva usando o método novo
	newCfg := config.NewConfig()
	if c.IssuesDir != "" {
		newCfg.IssuesDir = c.IssuesDir
	}
	if c.DeepSeekApiKey != "" {
		newCfg.DeepSeekApiKey = c.DeepSeekApiKey
	}
	return newCfg.Save()
}

func ResolveAPIKey() string {
	// Ordem: .env (via internal/config) > ~/.ds-ac/config.json
	c := config.NewConfig()
	if key := c.ResolveAPIKey(); key != "" {
		return key
	}
	if old, err := LoadConfig(); err == nil && old.DeepSeekApiKey != "" {
		return old.DeepSeekApiKey
	}
	return ""
}
