package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Mutex de pacote para evitar colisão E/S entre requisições concorrentes
var storeMu sync.RWMutex

func getChatsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "chats")
}

// sanitizeID bloqueia tentativas de Path Traversal
func sanitizeID(id string) (string, error) {
	clean := filepath.Clean(id)
	if clean == "." || clean == "/" || strings.Contains(clean, "..") || filepath.IsAbs(clean) || filepath.Base(clean) != id {
		return "", fmt.Errorf("ID de sessão inválido: %s", id)
	}
	return clean, nil
}

func SaveSession(s *ChatSession) error {
	if s == nil || s.ID == "" {
		return fmt.Errorf("sessão ou ID inválido")
	}

	cleanID, err := sanitizeID(s.ID)
	if err != nil {
		return err
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	dir := getChatsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}

	finalPath := filepath.Join(dir, cleanID+".json")
	tmpPath := filepath.Join(dir, "."+cleanID+".tmp")

	// Escrita atômica: grava no temporário e renomeia para evitar corrupção
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, finalPath)
}

func LoadSession(id string) (*ChatSession, error) {
	cleanID, err := sanitizeID(id)
	if err != nil {
		return nil, err
	}

	storeMu.RLock()
	defer storeMu.RUnlock()

	path := filepath.Join(getChatsDir(), cleanID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s ChatSession
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func ListSessions() ([]ChatSession, error) {
	storeMu.RLock()
	defer storeMu.RUnlock()

	dir := getChatsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChatSession{}, nil
		}
		return nil, err
	}

	var out []ChatSession
	for _, e := range entries {
		// Ignora diretórios, ficheiros que não sejam .json e temporários escondidos (.)
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || strings.HasPrefix(e.Name(), ".") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		var s ChatSession
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}

		if strings.TrimSpace(s.Title) == "" {
			title := ""
			for _, m := range s.Messages {
				if m.Role == "user" && m.Content != "" {
					title = m.Content
					break
				}
			}
			if len([]rune(title)) > 48 {
				r := []rune(title)
				title = string(r[:48]) + "…"
			}
			s.Title = title
		}
		s.BaseContent = ""
		s.ProjetoContent = ""
		s.Messages = nil
		out = append(out, s)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})

	return out, nil
}

func DeleteSession(id string) error {
	cleanID, err := sanitizeID(id)
	if err != nil {
		return err
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	path := filepath.Join(getChatsDir(), cleanID+".json")

	// Remoção direta e atômica, eliminando TOCTOU race condition
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("chat não encontrado")
		}
		return err
	}

	return nil
}

func RenameSession(id, title string) (*ChatSession, error) {
	s, err := LoadSession(id)
	if err != nil {
		return nil, err
	}
	s.Title = strings.TrimSpace(title)
	if err := SaveSession(s); err != nil {
		return nil, err
	}
	return s, nil
}

func BuildResumo(s *ChatSession) string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	title := s.Title
	if title == "" {
		title = "Chat"
	}
	b.WriteString("# "+title+"\n")
	if s.Project != "" {
		b.WriteString("Projeto: "+s.Project+"\n")
	}
	b.WriteString("Atualizado: "+s.UpdatedAt.Format("02/01/2006 15:04")+"\n\n")
	for _, m := range s.Messages {
		switch m.Role {
		case "user":
			b.WriteString("## Você\n\n"+m.Content+"\n\n")
		case "assistant":
			if strings.TrimSpace(m.Content) == "" {
				continue
			}
			b.WriteString("## Assistente\n\n"+m.Content+"\n\n")
		}
	}
	return b.String()
}

func MergeHistoryByProject() error {
	dir := getChatsDir()
	flag := filepath.Join(dir, ".merged-history")
	if _, err := os.Stat(flag); err == nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	groups := map[string][]ChatSession{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s ChatSession
		if json.Unmarshal(data, &s) != nil || s.ID == "" {
			continue
		}
		key := s.Project
		if key == "" {
			key = "_none"
		}
		groups[key] = append(groups[key], s)
	}
	for _, list := range groups {
		if len(list) < 2 {
			continue
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].CreatedAt.Equal(list[j].CreatedAt) {
				return list[i].ID < list[j].ID
			}
			return list[i].CreatedAt.Before(list[j].CreatedAt)
		})
		keep := list[0]
		if strings.TrimSpace(keep.Title) == "" {
			keep.Title = "Histórico"
		} else {
			keep.Title = "Histórico"
		}
		for _, extra := range list[1:] {
			for _, m := range extra.Messages {
				if m.Role == "system" {
					continue
				}
				keep.Messages = append(keep.Messages, m)
			}
			if extra.UpdatedAt.After(keep.UpdatedAt) {
				keep.UpdatedAt = extra.UpdatedAt
			}
			_ = os.Remove(filepath.Join(dir, extra.ID+".json"))
		}
		if err := SaveSession(&keep); err != nil {
			return err
		}
	}
	return os.WriteFile(flag, []byte("ok\n"), 0644)
}
