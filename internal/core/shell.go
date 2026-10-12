package core

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Configuração via variáveis de ambiente (todas opcionais):
//
//	DSAC_SHELL_ROOT     raiz máxima onde comandos podem rodar (padrão: $HOME/prj)
//	DSAC_SHELL_TIMEOUT  timeout em segundos (padrão: 120)
//	DSAC_SHELL_MODE     "allowlist" (padrão) ou "denylist"
//	DSAC_SHELL_ALLOW    lista extra de binários permitidos, separada por vírgula
const (
	shellDefaultTimeout = 120
	shellMaxOutput      = 64 * 1024
)

// Binários permitidos no modo allowlist. git é restrito a subcomandos de leitura.
var shellAllowed = map[string]bool{
	"go": true, "gofmt": true, "make": true, "dart": true, "flutter": true,
	"git": true, "ls": true, "cat": true, "grep": true, "rg": true, "tree": true,
	"head": true, "tail": true, "wc": true, "pwd": true, "diff": true,
}

var gitAllowedSub = map[string]bool{
	"status": true, "diff": true, "log": true, "show": true, "branch": true,
}

// Padrões bloqueados (best-effort) - valem nos dois modos.
var shellDenyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(^|[\s;&|(])(sudo|su|doas|pkexec)(\s|$)`),
	regexp.MustCompile(`(^|[\s;&|(])(mkfs[\w.]*|dd|shutdown|reboot|halt|poweroff|init|systemctl|crontab|useradd|userdel|passwd|mount|umount|chroot|nc|ncat|telnet)(\s|$)`),
	regexp.MustCompile(`rm\s+(-[a-zA-Z]*\s+)*(/|~|\$HOME|/\*)(\s|$)`),
	regexp.MustCompile(`(curl|wget)[^|]*\|\s*(ba|z|da)?sh`),
	regexp.MustCompile(`\.ssh|\.gnupg|\.aws|\.ds-ac|/etc/(passwd|shadow|sudoers)|\.env(\s|$|["'])`),
	regexp.MustCompile(`(chmod|chown)\s+(-[a-zA-Z]*R|--recursive)`),
	regexp.MustCompile(`:\(\)\s*\{`), // fork bomb
	regexp.MustCompile(`>\s*/dev/(sd|nvme|vd)`),
}

// ShellRoot retorna a raiz máxima permitida (resolvida, sem symlinks).
func ShellRoot() (string, error) {
	root := os.Getenv("DSAC_SHELL_ROOT")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, "prj")
	}
	return resolvePath(root)
}

func resolvePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

func isInside(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// RunShellCommand executa command no diretório projectRoot/cwd, validando
// que o diretório final está dentro de ShellRoot(). Retorna saída + exit code.
func RunShellCommand(projectRoot, cwd, command string) (string, error) {
	cmdStr := strings.TrimSpace(command)
	if cmdStr == "" {
		return "", fmt.Errorf("comando vazio informado")
	}

	root, err := ShellRoot()
	if err != nil {
		return "", fmt.Errorf("raiz de execução inválida: %v", err)
	}
	if cwd == "" {
		cwd = "."
	}
	dir := cwd
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(projectRoot, dir)
	}
	dir, err = resolvePath(dir)
	if err != nil {
		return "", fmt.Errorf("diretório inválido: %v", err)
	}
	if !isInside(root, dir) {
		return "", fmt.Errorf("diretório fora da raiz permitida (%s): %s", root, dir)
	}

	for _, re := range shellDenyPatterns {
		if re.MatchString(cmdStr) {
			log.Printf("[run_command] BLOQUEADO dir=%s cmd=%q padrão=%s", dir, cmdStr, re.String())
			return "", fmt.Errorf("comando bloqueado por política de segurança")
		}
	}

	timeout := shellDefaultTimeout
	if v, err := strconv.Atoi(os.Getenv("DSAC_SHELL_TIMEOUT")); err == nil && v > 0 {
		timeout = v
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if strings.EqualFold(os.Getenv("DSAC_SHELL_MODE"), "denylist") {
		// Modo denylist: shell completo, protegido apenas pelos padrões acima.
		cmd = exec.CommandContext(ctx, "bash", "-c", cmdStr)
	} else {
		// Modo allowlist: sem shell (sem pipes, redirecionamentos, encadeamento).
		args, err := validateAllowlist(cmdStr, root, dir)
		if err != nil {
			log.Printf("[run_command] BLOQUEADO dir=%s cmd=%q motivo=%v", dir, cmdStr, err)
			return "", err
		}
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}
	cmd.Dir = dir
	cmd.Env = minimalEnv(dir)

	var buf limitedBuffer
	buf.max = shellMaxOutput
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	log.Printf("[run_command] dir=%s timeout=%ds cmd=%q", dir, timeout, cmdStr)
	start := time.Now()
	runErr := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	log.Printf("[run_command] fim exit=%d dur=%s", exitCode, time.Since(start).Round(time.Millisecond))

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("exit=-1 (timeout %ds)\n%s", timeout, buf.String()), nil
	}
	if runErr != nil && cmd.ProcessState == nil {
		return "", fmt.Errorf("falha ao executar: %v", runErr)
	}
	return fmt.Sprintf("exit=%d\n%s", exitCode, buf.String()), nil
}

func validateAllowlist(cmdStr, root, dir string) ([]string, error) {
	if strings.ContainsAny(cmdStr, ";|&`$<>\n\\") {
		return nil, fmt.Errorf("operadores de shell não permitidos no modo allowlist (; | & ` $ < > \\)")
	}
	args := splitArgs(cmdStr)
	if len(args) == 0 {
		return nil, fmt.Errorf("comando inválido")
	}
	allowed := shellAllowed[args[0]]
	for _, extra := range strings.Split(os.Getenv("DSAC_SHELL_ALLOW"), ",") {
		if strings.TrimSpace(extra) == args[0] {
			allowed = true
		}
	}
	if !allowed {
		return nil, fmt.Errorf("comando não permitido: %s", args[0])
	}
	if args[0] == "git" && (len(args) < 2 || !gitAllowedSub[args[1]]) {
		return nil, fmt.Errorf("git permitido apenas com: status, diff, log, show, branch")
	}
	// Argumentos que parecem caminhos não podem sair da raiz.
	for _, a := range args[1:] {
		p := a
		if i := strings.Index(a, "="); i >= 0 && strings.HasPrefix(a, "-") {
			p = a[i+1:]
		}
		if strings.HasPrefix(p, "-") || p == "" {
			continue
		}
		if strings.HasPrefix(p, "~") {
			return nil, fmt.Errorf("argumento com ~ não permitido: %s", a)
		}
		if filepath.IsAbs(p) || strings.Contains(p, "..") {
			full := p
			if !filepath.IsAbs(full) {
				full = filepath.Join(dir, full)
			}
			if !isInside(root, filepath.Clean(full)) {
				return nil, fmt.Errorf("argumento aponta para fora da raiz: %s", a)
			}
		}
	}
	return args, nil
}

// splitArgs separa argumentos respeitando aspas simples e duplas.
func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	var quote rune
	inArg := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inArg = true
		case r == ' ' || r == '\t':
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if inArg {
		args = append(args, cur.String())
	}
	return args
}

// minimalEnv repassa só o necessário para toolchains; nunca chaves de API do .env.
func minimalEnv(dir string) []string {
	keep := []string{"PATH", "HOME", "USER", "LANG", "LC_ALL", "TERM", "TMPDIR",
		"GOPATH", "GOROOT", "GOCACHE", "GOMODCACHE", "GOFLAGS", "GOPROXY", "GOPRIVATE",
		"FLUTTER_ROOT", "PUB_CACHE", "ANDROID_HOME", "ANDROID_SDK_ROOT", "JAVA_HOME"}
	env := []string{"PWD=" + dir}
	for _, k := range keep {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}

// limitedBuffer guarda no máximo max bytes e descarta o resto.
type limitedBuffer struct {
	buf       bytes.Buffer
	max       int
	truncated int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remain := b.max - b.buf.Len()
	if remain <= 0 {
		b.truncated += len(p)
		return len(p), nil
	}
	if len(p) > remain {
		b.buf.Write(p[:remain])
		b.truncated += len(p) - remain
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string {
	if b.truncated > 0 {
		return b.buf.String() + fmt.Sprintf("\n... [saída truncada: %d bytes descartados]", b.truncated)
	}
	return b.buf.String()
}
