package chat

import (
	"time"

	"github.com/google/uuid"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ChatSession guarda o histórico. Usa dto.Message como base, não redefine Message/ToolCall.
type ChatSession struct {
	ID             string        `json:"id"`
	Project        string        `json:"project"`
	Title          string        `json:"title,omitempty"`
	BasePath       string        `json:"base_path"`
	ProjetoPath    string        `json:"projeto_path"`
	BaseContent    string        `json:"base_content,omitempty"`
	ProjetoContent string        `json:"projeto_content,omitempty"`
	Messages       []dto.Message `json:"messages"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Session mantém compatibilidade com código antigo que usa chat.Session
type Session = ChatSession

func NewSession(id, project, basePath, projetoPath, baseContent, projetoContent string) *ChatSession {
	now := time.Now()
	return &ChatSession{
		ID:             id,
		Project:        project,
		BasePath:       basePath,
		ProjetoPath:    projetoPath,
		BaseContent:    baseContent,
		ProjetoContent: projetoContent,
		Title:          "Novo chat",
		Messages:       []dto.Message{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func NewSessionWithID(project string) *ChatSession {
	return NewSession(uuid.New().String(), project, "", "", "", "")
}
