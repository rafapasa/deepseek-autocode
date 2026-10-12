package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Nomes sensíveis bloqueados para leitura/escrita pelas tools do LLM.
var sensitiveNames = map[string]bool{
	".env": true, ".ssh": true, ".gnupg": true, ".aws": true, ".ds-ac": true,
}

func isSensitive(p string) bool {
	for _, part := range strings.Split(filepath.ToSlash(p), "/") {
		if sensitiveNames[part] {
			return true
		}
		if strings.HasPrefix(part, ".env.") && part != ".env.example" {
			return true
		}
	}
	return false
}

// resolveExisting resolve symlinks do caminho; se ele (ou parte dele) ainda não
// existe, resolve o ancestral existente mais próximo e reanexa o restante.
func resolveExisting(abs string) (string, error) {
	cur := abs
	var rest []string
	for {
		r, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return filepath.Join(append([]string{r}, rest...)...), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", err
		}
		rest = append([]string{filepath.Base(cur)}, rest...)
		cur = parent
	}
}

// ResolveSafePath resolve p (relativo a base, ou absoluto) e garante que o
// destino real (após symlinks) fica dentro de ShellRoot() e não é sensível.
func ResolveSafePath(base, p string) (string, error) {
	root, err := ShellRoot()
	if err != nil {
		return "", fmt.Errorf("raiz permitida inválida: %v", err)
	}
	if p == "" {
		p = "."
	}
	if isSensitive(p) {
		return "", fmt.Errorf("acesso a arquivo sensível não permitido: %s", p)
	}
	full := p
	if !filepath.IsAbs(full) {
		full = filepath.Join(base, full)
	}
	full, err = filepath.Abs(full)
	if err != nil {
		return "", err
	}
	resolved, err := resolveExisting(full)
	if err != nil {
		return "", fmt.Errorf("caminho inválido: %v", err)
	}
	if !isInside(root, resolved) {
		return "", fmt.Errorf("caminho fora da raiz permitida (%s): %s", root, p)
	}
	if isSensitive(resolved) {
		return "", fmt.Errorf("acesso a arquivo sensível não permitido: %s", p)
	}
	return resolved, nil
}
