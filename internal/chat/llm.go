package chat

import (
	"github.com/rafapasa/deepseek-autocode/internal/config"
	"github.com/rafapasa/deepseek-autocode/internal/core"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

// ChatLLM é só um wrapper - não tem URL, model ou key hardcodado.
// Toda lógica de LLM está no core.
type ChatLLM struct {
	cfg    *config.Config
	client core.LlmInterface
}

func NewChatLLM(cfg *config.Config) *ChatLLM {
	return &ChatLLM{
		cfg:    cfg,
		client: core.NewLlmClient(*cfg),
	}
}

func (c *ChatLLM) BuildSystemPrompt(baseContent, projetoContent, projectRoot string) string {
	return core.BuildSystemPrompt(baseContent, projetoContent, projectRoot)
}

func (c *ChatLLM) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	return c.client.Chat(messages, tools)
}

func (c *ChatLLM) StreamChat(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error {
	return c.client.ChatStream(messages, tools, onDelta, onToolCall)
}
