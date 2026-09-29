package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rafapasa/deepseek-autocode/internal/config"
)

type Run struct {
	ID       string
	Project  string
	Issue    string
	History  []string
	Notify   chan struct{}
	Success  bool
	Finished bool
	Err      string
	PID      int
	Started  time.Time
	cancel   context.CancelFunc
	mu       sync.Mutex
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
		History: make([]string, 0, 512),
		Notify:  make(chan struct{}, 8),
		Started: time.Now(),
		cancel:  cancel,
	}
	runs.Store(id, r)
	go r.exec(ctx, cfg, issuePath)
	return r, nil
}

func (r *Run) appendLine(line string) {
	r.mu.Lock()
	r.History = append(r.History, line)
	r.mu.Unlock()
	select {
	case r.Notify <- struct{}{}:
	default:
	}
}

func (r *Run) snapshot(from int) (lines []string, finished, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if from < 0 {
		from = 0
	}
	if from > len(r.History) {
		from = len(r.History)
	}
	out := append([]string(nil), r.History[from:]...)
	return out, r.Finished, r.Success
}

func (r *Run) WaitLogs(from int, wait time.Duration) (lines []string, next int, finished, success bool) {
	lines, finished, success = r.snapshot(from)
	if len(lines) > 0 || finished || wait <= 0 {
		return lines, from + len(lines), finished, success
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for {
		select {
		case <-r.Notify:
			lines, finished, success = r.snapshot(from)
			if len(lines) > 0 || finished {
				return lines, from + len(lines), finished, success
			}
		case <-timer.C:
			lines, finished, success = r.snapshot(from)
			return lines, from + len(lines), finished, success
		}
	}
}

func (r *Run) exec(ctx context.Context, cfg *config.Config, issuePath string) {
	defer func() {
		r.mu.Lock()
		r.Finished = true
		r.mu.Unlock()
		select {
		case r.Notify <- struct{}{}:
		default:
		}
		r.Cleanup()
	}()

	binPath, err := os.Executable()
	if err != nil {
		r.appendLine("❌ erro ao localizar binário: " + err.Error())
		return
	}

	args := []string{}
	if key := cfg.ResolveAPIKey(); key != "" {
		args = append(args, "--key", key)
	}
	args = append(args, issuePath)

	r.appendLine(fmt.Sprintf("[ui] executando %s %s", filepath.Base(binPath), issuePath))

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Env = os.Environ()
	cmd.Dir = filepath.Dir(binPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.appendLine("❌ erro no pipe stdout: " + err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.appendLine("❌ erro no pipe stderr: " + err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		r.appendLine("❌ erro ao iniciar processo: " + err.Error())
		r.mu.Lock()
		r.Err = err.Error()
		r.mu.Unlock()
		return
	}

	r.mu.Lock()
	r.PID = cmd.Process.Pid
	r.mu.Unlock()
	r.appendLine(fmt.Sprintf("[ui] pid %d iniciado", cmd.Process.Pid))

	var wg sync.WaitGroup
	pump := func(rd io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(rd)
		sc.Buffer(make([]byte, 1024*1024), 2*1024*1024)
		for sc.Scan() {
			r.appendLine(sc.Text())
		}
		if err := sc.Err(); err != nil && err != io.EOF {
			r.appendLine("⚠ leitura do processo: " + err.Error())
		}
	}
	wg.Add(2)
	go pump(stdout)
	go pump(stderr)

	err = cmd.Wait()
	wg.Wait()

	success := err == nil
	if err != nil {
		msg := err.Error()
		r.mu.Lock()
		r.Err = msg
		r.mu.Unlock()
		r.appendLine("⚠ processo terminou com erro: " + msg)
	} else {
		r.appendLine("[ui] processo finalizado com sucesso")
	}

	if success {
		r.moveToConcluidas(issuePath)
	}

	r.mu.Lock()
	r.Success = success
	r.mu.Unlock()
}

func (r *Run) moveToConcluidas(issuePath string) {
	dir := filepath.Dir(issuePath)
	concluidas := filepath.Join(dir, "concluidas")
	if err := os.MkdirAll(concluidas, 0755); err != nil {
		r.appendLine("⚠ não foi possível criar pasta de concluídas: " + err.Error())
		return
	}

	base := filepath.Base(issuePath)
	target := filepath.Join(concluidas, base)
	if err := os.Rename(issuePath, target); err != nil {
		r.appendLine("⚠ não foi possível mover a issue: " + err.Error())
		return
	}

	logSrc := issuePath + ".log"
	if _, err := os.Stat(logSrc); err == nil {
		_ = os.Rename(logSrc, target+".log")
	}

	r.appendLine("📦 issue movida para concluidas/" + base)
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
	time.AfterFunc(2*time.Hour, func() { runs.Delete(r.ID) })
}
