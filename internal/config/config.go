package config

type Config struct {
	DeepSeekApiKey string,
	MetaApiKey string,
}

func NewConfig(deepSeekApiKey, metaApiKey string) *Config {
	os.
	return &Config{
			DeepSeekApiKey: os.Setenv("DEEPSEEK_API_KEY", key),
			MetaApiKey: os.Setenv("META_API_KEY", key),
		}
}
