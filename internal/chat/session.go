package chat

import (
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	ID         string `json:"id"`
	Role       Role   `json:"role"`
	Content    string `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Name       string `json:"name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // function
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type ChatSession struct {
	ID            string    `json:"id"`
	Project       string    `json:"project"`
	BasePath      string    `json:"base_path"`
	ProjetoPath   string    `json:"projeto_path"`
	BaseContent   string    `json:"base_content,omitempty"`
	ProjetoContent string   `json:"projeto_content,omitempty"`
	Messages      []Message `json:"messages"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewSession(id, project, basePath, projetoPath, baseContent, projetoContent string) *ChatSession {
	now := time.Now()
	return &ChatSession{
		ID:             id,
		Project:        project,
		BasePath:       basePath,
		ProjetoPath:    projetoPath,
		BaseContent:    baseContent,
		ProjetoContent: projetoContent,
		Messages:       []Message{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

