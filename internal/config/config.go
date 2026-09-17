package config

type Config struct {
	DeepSeekApiKey string
}

func NewConfig(deepSeekApiKey string) *Config {
	return &Config{DeepSeekApiKey: deepSeekApiKey}
}
