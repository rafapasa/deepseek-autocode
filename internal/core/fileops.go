package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SafeJoin junta raiz + rel, bloqueando path traversal.
func SafeJoin(raiz, rel string) (string, error) {
	clean := filepath.Clean(rel)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("caminho não permitido: %s", rel)
	}
	return filepath.Join(raiz, clean), nil
}
