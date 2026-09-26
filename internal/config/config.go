package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	LLM_DEEPSEEK = 1
	LLM_META     = 2
)

type Config struct {
	LlmClient      int
	DeepSeekApiKey string
	MetaApiKey     string
}

func NewConfig() *Config {
	_ = godotenv.Load()
	return &Config{
		LlmClient:      getEnvAsInt("LLM_CLIENT", LLM_DEEPSEEK),
		DeepSeekApiKey: getEnv("DEEPSEEK_API_KEY", ""),
		MetaApiKey:     getEnv("META_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
