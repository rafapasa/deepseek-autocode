package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	IssuesDir string `json:"issues_dir"`
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
