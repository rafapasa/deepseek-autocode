package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSafePath(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "prj")
	proj := filepath.Join(root, "proj")
	outside := filepath.Join(tmp, "fora")
	for _, d := range []string{proj, outside} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(proj, "link")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSAC_SHELL_ROOT", root)

	permitidos := []string{"main.go", "novo/dir/arquivo.go", ".", filepath.Join(proj, "x.go"), "../outro/a.go", ".env.example"}
	for _, p := range permitidos {
		if _, err := ResolveSafePath(proj, p); err != nil {
			t.Errorf("deveria permitir %q: %v", p, err)
		}
	}

	bloqueados := []string{
		"/etc/passwd", outside, "../../fora/x", "../..",
		"link/segredo.txt", "link/novo/arquivo.txt",
		".env", "sub/.env", ".env.local", ".ssh/id_rsa", ".aws/credentials", ".gnupg", ".ds-ac/config.json",
	}
	for _, p := range bloqueados {
		if _, err := ResolveSafePath(proj, p); err == nil {
			t.Errorf("deveria bloquear %q", p)
		}
	}

	// symlink interno apontando para .env deve ser bloqueado
	envPath := filepath.Join(proj, ".env")
	_ = os.WriteFile(envPath, []byte("X=1"), 0600)
	_ = os.Symlink(envPath, filepath.Join(proj, "inocente.txt"))
	if _, err := ResolveSafePath(proj, "inocente.txt"); err == nil {
		t.Error("symlink para .env deveria ser bloqueado")
	}
}
