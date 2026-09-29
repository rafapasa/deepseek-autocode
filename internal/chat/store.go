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
