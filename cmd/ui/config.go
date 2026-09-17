package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
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
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ResolveAPIKey retorna a chave em ordem de prioridade:
//  1. flag --key (passada pelo CLI, chegou via env)
//  2. config.json (~/.ds-ac/config.json)
//  3. env DEEPSEEK_API_KEY
func ResolveAPIKey() string {
	// 1. env primeiro (a flag já foi propagada pro env pelo main.go)
	if k := os.Getenv("DEEPSEEK_API_KEY"); k != "" {
		return k
	}
	// 2. config
	if c, err := LoadConfig(); err == nil && c.DeepSeekApiKey != "" {
		return c.DeepSeekApiKey
	}
	return ""
}
