package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunShellCommand(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "proj")
	if err := os.MkdirAll(proj, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DSAC_SHELL_ROOT", root)
	t.Setenv("DSAC_SHELL_MODE", "")
	t.Setenv("DEEPSEEK_API_KEY", "segredo")

	out, err := RunShellCommand(proj, ".", "ls")
	if err != nil || !strings.HasPrefix(out, "exit=0") {
		t.Fatalf("ls deveria funcionar: %v %q", err, out)
	}

	bloqueados := []string{
		"sudo ls", "rm -rf /", "bash -c ls", "ls | sh", "ls; ls",
		"cat /etc/passwd", "cat ../../x", "git push", "cat .env", "ls ~/.ssh",
	}
	for _, c := range bloqueados {
		if _, err := RunShellCommand(proj, ".", c); err == nil {
			t.Errorf("deveria bloquear: %q", c)
		}
	}

	if _, err := RunShellCommand(proj, "/tmp", "ls"); err == nil {
		t.Error("cwd fora da raiz deveria ser bloqueado")
	}

	t.Setenv("DSAC_SHELL_MODE", "denylist")
	out, err = RunShellCommand(proj, ".", "env")
	if err != nil || strings.Contains(out, "segredo") {
		t.Errorf("env do filho não deve conter segredos: %v %q", err, out)
	}
}
