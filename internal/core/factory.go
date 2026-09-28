package core

import "github.com/rafapasa/deepseek-autocode/internal/config"

func NewLlmClient(cfg config.Config) LlmInterface {
	switch cfg.LlmClient {
	case config.LLM_DEEPSEEK:
		return NewDeepSeekClient(cfg.DeepSeekApiKey)
	case config.LLM_META:
		return NewLlamaClient(cfg.MetaApiKey)
	default:
		if cfg.MetaApiKey != "" {
			return NewLlamaClient(cfg.MetaApiKey)
		}
		return NewDeepSeekClient(cfg.DeepSeekApiKey)
	}
}
