package core

import "github.com/rafapasa/deepseek-autocode/internal/dto"

type LlmInterface interface {
	Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error)
}