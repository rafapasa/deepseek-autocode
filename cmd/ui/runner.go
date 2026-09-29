package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rafapasa/deepseek-autocode/internal/config"
)

type Run struct {
	ID      string
	Project string
	Issue   string
	Lines   chan string
	Done    chan bool
	Success bool
	cancel  context.CancelFunc
	mu      sync.Mutex
}

var (
	runs   sync.Map
	runSeq int64
)

func StartRun(cfg *config.Config, project, issue string) (*Run, error) {
	id := fmt.Sprintf("r%d", atomic.AddInt64(&runSeq, 1))

	if cfg.IssuesDir == "" {
		return nil, fmt.Errorf("issues_dir não configurado")
	}

	issuePath := filepath.Join(cfg.IssuesDir, project, issue)
	if _, err := os.Stat(issuePath); err != nil {
		return nil, fmt.Errorf("issue não encontrada: %s", issuePath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Run{
		ID:      id,
		Project: project,
		Issue:   issue,
		Lines:   make(chan string, 200),
		Done:    make(chan bool, 1),
		cancel:  cancel,
	}
	runs.Store(id, r)

	go r.exec(ctx, cfg, issuePath)
	return r, nil
}

func (r *Run) exec(ctx context.Context, cfg *config.Config, issuePath string) {
	defer close(r.Lines)
	defer close(r.Done)

	binPath, err := os.Executable()
	if err != nil {
		r.safeSend("❌ erro ao localizar binário: " + err.Error())
		r.Done <- false
		return
	}

	args := []string{}
	if key := cfg.ResolveAPIKey(); key != "" {
		args = append(args, "--key", key)
	}
	args = append(args, issuePath)

	cmd := exec.CommandContext(ctx, binPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.safeSend("❌ erro no pipe: " + err.Error())
		r.Done <- false
		return
	}
	cmd.Stderr = cmd.Stdout
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		r.safeSend("❌ erro ao iniciar: " + err.Error())
		r.Done <- false
		return
	}

	// Espera scanner terminar antes de fechar canal
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			text := scanner.Text()
			// Não bloqueia se contexto cancelado
			select {
			case <-ctx.Done():
				return
			case r.Lines <- text:
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			log.Printf("scanner erro: %v", err)
		}
	}()

	err = cmd.Wait()
	wg.Wait() // garante que todo stdout foi enviado antes de fechar

	success := err == nil
	if err != nil {
		r.safeSend("⚠ processo terminou com erro: " + err.Error())
	}

	if success {
		r.moveToConcluidas(issuePath)
	}

	r.mu.Lock()
	r.Success = success
	r.mu.Unlock()
	r.Done <- success
}

func (r *Run) safeSend(line string) {
	defer func() {
		if rec := recover(); rec != nil {
			// canal já fechado, ignora
			log.Printf("safeSend recover: %v", rec)
		}
	}()
	select {
	case r.Lines <- line:
	default:
		// buffer cheio, tenta com timeout curto
		select {
		case r.Lines <- line:
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (r *Run) moveToConcluidas(issuePath string) {
	dir := filepath.Dir(issuePath)
	concluidas := filepath.Join(dir, "concluidas")
	if err := os.MkdirAll(concluidas, 0755); err != nil {
		r.safeSend("⚠ não foi possível criar pasta de concluídas: " + err.Error())
		return
	}

	base := filepath.Base(issuePath)
	target := filepath.Join(concluidas, base)
	if err := os.Rename(issuePath, target); err != nil {
		r.safeSend("⚠ não foi possível mover a issue: " + err.Error())
		return
	}

	logSrc := issuePath + ".log"
	if _, err := os.Stat(logSrc); err == nil {
		_ = os.Rename(logSrc, target+".log")
	}

	r.safeSend("📦 issue movida para concluidas/" + base)
}

func (r *Run) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
}

func GetRun(id string) (*Run, bool) {
	v, ok := runs.Load(id)
	if !ok {
		return nil, false
	}
	return v.(*Run), true
}

func (r *Run) Cleanup() {
	time.AfterFunc(5*time.Minute, func() { runs.Delete(r.ID) })
}

var _ = io.Discard
