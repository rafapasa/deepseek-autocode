package chat

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rafapasa/deepseek-autocode/internal/config"
	"github.com/rafapasa/deepseek-autocode/internal/dto"
)

type Service struct {
	cfg *config.Config
	llm *ChatLLM
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg, llm: NewChatLLM(cfg)}
}

func (s *Service) CreateSession(project string) (*ChatSession, error) {
	id := uuid.New().String()
	basePath := s.cfg.BaseJsonPath
	projetoPath := s.cfg.ProjetoJsonPath
	if project != "" && s.cfg.IssuesDir != "" {
		candidate := filepath.Join(s.cfg.IssuesDir, project, "projeto.json")
		if _, err := os.Stat(candidate); err == nil {
			projetoPath = candidate
		}
	}
	baseContent := ""
	if data, err := os.ReadFile(basePath); err == nil {
		baseContent = string(data)
	}
	projetoContent := ""
	if projetoPath != "" {
		if data, err := os.ReadFile(projetoPath); err == nil {
			projetoContent = string(data)
		}
	}
	session := NewSession(id, project, basePath, projetoPath, baseContent, projetoContent)
	projectRoot := s.ResolveProjectRoot(project)
	sysPrompt := s.llm.BuildSystemPrompt(baseContent, projetoContent, projectRoot)
	session.Messages = append(session.Messages, dto.Message{Role: "system", Content: sysPrompt})
	if err := SaveSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) AddUserMessage(sessionID, content string) (*ChatSession, error) {
	session, err := LoadSession(sessionID)
	if err != nil {
		return nil, err
	}
	session.Messages = append(session.Messages, dto.Message{Role: "user", Content: content})
	session.UpdatedAt = time.Now()
	return session, SaveSession(session)
}

func (s *Service) AddAssistantMessage(sessionID, content string) error {
	session, err := LoadSession(sessionID)
	if err != nil {
		return err
	}
	session.Messages = append(session.Messages, dto.Message{Role: "assistant", Content: content})
	session.UpdatedAt = time.Now()
	return SaveSession(session)
}

func (s *Service) AddToolMessage(sessionID, toolCallID, content string) error {
	session, err := LoadSession(sessionID)
	if err != nil {
		return err
	}
	session.Messages = append(session.Messages, dto.Message{Role: "tool", ToolCallID: toolCallID, Content: content})
	session.UpdatedAt = time.Now()
	return SaveSession(session)
}

func (s *Service) GetLLMMessages(sessionID string) ([]dto.Message, error) {
	session, err := LoadSession(sessionID)
	if err != nil {
		return nil, err
	}
	var out []dto.Message
	for _, m := range session.Messages {
		if m.Role == "system" || m.Role == "user" || m.Role == "assistant" || m.Role == "tool" {
			out = append(out, dto.Message{Role: m.Role, Content: m.Content, ToolCalls: m.ToolCalls, ToolCallID: m.ToolCallID, Name: m.Name})
		}
	}
	return out, nil
}

func (s *Service) ResolveProjectRoot(project string) string {
	if s.cfg.IssuesDir == "" {
		return "."
	}
	if project == "" {
		return s.cfg.IssuesDir
	}
	candidate := filepath.Join(filepath.Dir(s.cfg.IssuesDir), project)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return filepath.Join(s.cfg.IssuesDir, project)
}

func (s *Service) ExportToIssue(sessionID, demanda string) (string, error) {
	session, err := LoadSession(sessionID)
	if err != nil {
		return "", err
	}
	if s.cfg.IssuesDir == "" {
		return "", fmt.Errorf("ISSUES_DIR vazio")
	}
	projectDir := filepath.Join(s.cfg.IssuesDir, session.Project)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		log.Printf("Erro handler.ExportToIssue Mkdir: %v", err)
	}
	last := ""
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Role == "assistant" {
			last = session.Messages[i].Content
			break
		}
	}
	if demanda == "" {
		demanda = last
	}
	filename := fmt.Sprintf("issue-chat-%s.json", time.Now().Format("20060102-150405"))
	path := filepath.Join(projectDir, filename)
	content := fmt.Sprintf(`{"demanda": %q, "raiz": "%s", "arquivos": [], "tarefas": [{"descricao": "via eTools-Code chat %s"}] }`, demanda, session.Project, session.ID)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}
