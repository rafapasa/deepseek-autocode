package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func getChatsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "chats")
}

func SaveSession(s *ChatSession) error {
	dir := getChatsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, s.ID+".json")
	data, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadSession(id string) (*ChatSession, error) {
	path := filepath.Join(getChatsDir(), id+".json")
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
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
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
	path := filepath.Join(getChatsDir(), id+".json")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("chat não encontrado")
	}
	return os.Remove(path)
}
