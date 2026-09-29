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
	basePath, projetoPath, baseContent, projetoContent := s.loadContextFiles(project)
	session := NewSession(id, project, basePath, projetoPath, baseContent, projetoContent)
	projectRoot := s.ResolveProjectRoot(project)
	sysPrompt := s.llm.BuildSystemPrompt(baseContent, projetoContent, projectRoot)
	session.Messages = append(session.Messages, dto.Message{Role: "system", Content: sysPrompt})
	if err := SaveSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) loadContextFiles(project string) (basePath, projetoPath, baseContent, projetoContent string) {
	basePath = s.cfg.BaseJsonPath
	if basePath == "" && s.cfg.IssuesDir != "" {
		basePath = filepath.Join(s.cfg.IssuesDir, "base.json")
	}
	projetoPath = s.cfg.ProjetoJsonPath
	if project != "" && s.cfg.IssuesDir != "" {
		candidate := filepath.Join(s.cfg.IssuesDir, project, "projeto.json")
		if _, err := os.Stat(candidate); err == nil {
			projetoPath = candidate
		}
	}
	if data, err := os.ReadFile(basePath); err == nil {
		baseContent = string(data)
	}
	if projetoPath != "" {
		if data, err := os.ReadFile(projetoPath); err == nil {
			projetoContent = string(data)
		}
	}
	return
}

func (s *Service) RefreshContext(session *ChatSession) {
	if session == nil {
		return
	}
	basePath, projetoPath, baseContent, projetoContent := s.loadContextFiles(session.Project)
	session.BasePath = basePath
	session.ProjetoPath = projetoPath
	session.BaseContent = baseContent
	session.ProjetoContent = projetoContent
	sysPrompt := s.llm.BuildSystemPrompt(baseContent, projetoContent, s.ResolveProjectRoot(session.Project))
	if len(session.Messages) == 0 || session.Messages[0].Role != "system" {
		session.Messages = append([]dto.Message{{Role: "system", Content: sysPrompt}}, session.Messages...)
		return
	}
	session.Messages[0].Content = sysPrompt
}

func (s *Service) AddUserMessage(sessionID, content string) (*ChatSession, error) {
	session, err := LoadSession(sessionID)
	if err != nil {
		return nil, err
	}
	s.RefreshContext(session)
	session.Messages = append(session.Messages, dto.Message{Role: "user", Content: content})
	session.UpdatedAt = time.Now()
	if err := SaveSession(session); err != nil {
		return nil, err
	}
	return session, nil
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

func (s *Service) PersistTurn(session *ChatSession) error {
	if session == nil {
		return fmt.Errorf("sessão nula")
	}
	session.UpdatedAt = time.Now()
	return SaveSession(session)
}

func SanitizeMessages(msgs []dto.Message) []dto.Message {
	var out []dto.Message
	pending := map[string]struct{}{}
	flushIncomplete := func() {
		if len(pending) == 0 {
			return
		}
		// remove assistant com tool_calls sem todas as respostas + tools órfãos do final
		for i := len(out) - 1; i >= 0; i-- {
			m := out[i]
			if m.Role == "tool" {
				out = out[:i]
				continue
			}
			if m.Role == "assistant" && len(m.ToolCalls) > 0 {
				out = out[:i]
			}
			break
		}
		pending = map[string]struct{}{}
	}
	for _, m := range msgs {
		switch m.Role {
		case "system", "user":
			flushIncomplete()
			out = append(out, m)
		case "assistant":
			flushIncomplete()
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					if tc.ID != "" {
						pending[tc.ID] = struct{}{}
					}
				}
			}
			out = append(out, m)
		case "tool":
			if _, ok := pending[m.ToolCallID]; ok {
				out = append(out, m)
				delete(pending, m.ToolCallID)
			}
		}
	}
	flushIncomplete()
	return out
}

func (s *Service) GetLLMMessages(sessionID string) ([]dto.Message, error) {
	session, err := LoadSession(sessionID)
	if err != nil {
		return nil, err
	}
	return SanitizeMessages(session.Messages), nil
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
		if session.Messages[i].Role == "assistant" && session.Messages[i].Content != "" {
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
