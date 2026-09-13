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

func StartRun(issuesDir, project, issue string) (*Run, error) {
	id := fmt.Sprintf("r%d", atomic.AddInt64(&runSeq, 1))

	issuePath := filepath.Join(issuesDir, project, issue)
	if _, err := os.Stat(issuePath); err != nil {
		return nil, fmt.Errorf("issue não encontrada: %s", issuePath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Run{
		ID:      id,
		Project: project,
		Issue:   issue,
		Lines:   make(chan string, 100),
		Done:    make(chan bool, 1),
		cancel:  cancel,
	}
	runs.Store(id, r)

	go r.exec(ctx, issuePath)
	return r, nil
}

func (r *Run) exec(ctx context.Context, issuePath string) {
	defer close(r.Lines)
	defer close(r.Done)

	binPath, err := os.Executable()
	if err != nil {
		r.Lines <- "❌ erro ao localizar binário: " + err.Error()
		r.Done <- false
		return
	}

	cmd := exec.CommandContext(ctx, binPath, issuePath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.Lines <- "❌ erro no pipe: " + err.Error()
		r.Done <- false
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		r.Lines <- "❌ erro ao iniciar: " + err.Error()
		r.Done <- false
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			r.Lines <- scanner.Text()
		}
	}()

	err = cmd.Wait()
	success := err == nil

	if success {
		r.moveToConcluidas(issuePath)
	}

	r.Success = success
	r.Done <- success
}

func (r *Run) moveToConcluidas(issuePath string) {
	dir := filepath.Dir(issuePath)
	concluidas := filepath.Join(dir, "concluidas")
	if err := os.MkdirAll(concluidas, 0755); err != nil {
		r.Lines <- "⚠ não foi possível criar pasta de concluídas: " + err.Error()
		return
	}

	base := filepath.Base(issuePath)
	target := filepath.Join(concluidas, base)
	if err := os.Rename(issuePath, target); err != nil {
		r.Lines <- "⚠ não foi possível mover a issue: " + err.Error()
		return
	}

	// move o log também, se existir
	logSrc := issuePath + ".log"
	if _, err := os.Stat(logSrc); err == nil {
		_ = os.Rename(logSrc, target+".log")
	}

	r.Lines <- "📦 issue movida para concluidas/" + base
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
