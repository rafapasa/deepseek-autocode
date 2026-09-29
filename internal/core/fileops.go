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

// SafeJoin junta raiz + rel garantindo estritamente que o resultado fique dentro de raiz.
func SafeJoin(raiz, rel string) (string, error) {
	if rel == "" {
		rel = "."
	}
	cleanRaiz := filepath.Clean(raiz)
	targetPath := filepath.Clean(filepath.Join(cleanRaiz, rel))

	// filepath.Rel valida se targetPath não tenta subir além de cleanRaiz
	relPath, err := filepath.Rel(cleanRaiz, targetPath)
	if err != nil || strings.HasPrefix(relPath, "..") || relPath == ".." {
		return "", fmt.Errorf("caminho não permitido (fora da raiz): %s", rel)
	}

	return targetPath, nil
}
