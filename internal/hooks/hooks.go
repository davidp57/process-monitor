// Package hooks runs the actions declared around a session: copies,
// deletions, commands, and sync pairs (copied in before, back after).
package hooks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/logx"
)

// Runner executes hooks and logs each step.
type Runner struct {
	Log    *logx.Logger
	Backup bool
	// Timeout bounds each exec action; 0 means no limit.
	Timeout time.Duration
}

// snapshot is a live file as it was before the profile was copied over it.
type snapshot struct {
	path   string
	data   []byte // nil: the file did not exist
	mod    time.Time
	exists bool
}

// Before copies every sync pair in (profile -> live), then runs the before
// actions. It stops at the first failure and puts the live files back as they
// were: launching with half a profile in place is worse than not launching,
// and leaving half a profile behind is no better.
func (r *Runner) Before(ctx context.Context, c *config.Config) error {
	var done []snapshot
	fail := func(err error) error {
		for i := len(done) - 1; i >= 0; i-- {
			r.restore(done[i])
		}
		return err
	}
	for _, p := range c.Sync {
		snap, err := take(p.Target)
		if err != nil {
			r.Log.Errorf("lecture de %s avant copie : %v", p.Target, err)
			return fail(err)
		}
		if err := r.copyFile(p.Source, p.Target); err != nil {
			return fail(err)
		}
		done = append(done, snap)
	}
	for _, a := range c.Hooks.Before {
		if err := r.run(ctx, a); err != nil {
			return fail(err)
		}
	}
	return nil
}

func take(path string) (snapshot, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot{path: path}, nil
	}
	if err != nil {
		return snapshot{}, err
	}
	data, err := os.ReadFile(path)
	return snapshot{path: path, data: data, mod: info.ModTime(), exists: true}, err
}

func (r *Runner) restore(s snapshot) {
	var err error
	if s.exists {
		err = writeAtomic(s.path, s.data, s.mod)
	} else {
		err = os.Remove(s.path)
	}
	if err != nil {
		r.Log.Errorf("remise en état de %s : %v", s.path, err)
		return
	}
	r.Log.Infof("remis en état : %s", s.path)
}

// After copies every sync pair back (live -> profile), then runs the after
// actions. It runs everything and returns all failures: one missing file
// must not stop the others from being saved.
func (r *Runner) After(ctx context.Context, c *config.Config) error {
	var errs []error
	for _, p := range c.Sync {
		errs = append(errs, r.copyFile(p.Target, p.Source))
	}
	for _, a := range c.Hooks.After {
		errs = append(errs, r.run(ctx, a))
	}
	return errors.Join(errs...)
}

func (r *Runner) run(ctx context.Context, a config.Action) error {
	switch {
	case a.Copy != nil:
		return r.copyFile(a.Copy.From, a.Copy.To)
	case a.Delete != "":
		return r.delete(a.Delete)
	default:
		return r.exec(ctx, a.Exec)
	}
}

func (r *Runner) copyFile(from, to string) error {
	if err := copyFile(from, to, r.Backup); err != nil {
		r.Log.Errorf("copie %s -> %s : %v", from, to, err)
		return err
	}
	r.Log.Infof("copié %s -> %s", from, to)
	return nil
}

func (r *Runner) delete(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		r.Log.Infof("rien à supprimer : %s", path)
		return nil
	}
	if err != nil {
		r.Log.Errorf("suppression %s : %v", path, err)
		return err
	}
	if info.IsDir() {
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}
	if err != nil {
		r.Log.Errorf("suppression %s : %v", path, err)
		return err
	}
	r.Log.Infof("supprimé %s", path)
	return nil
}

func (r *Runner) exec(ctx context.Context, argv []string) error {
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	// A program the command started in the background keeps the output
	// open; do not wait for it once the command itself has finished.
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.CombinedOutput()
	if errors.Is(err, exec.ErrWaitDelay) {
		err = nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("délai de %s dépassé", r.Timeout)
	}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\r\n"), "\n") {
		if line != "" {
			r.Log.Output(strings.TrimRight(line, "\r"))
		}
	}
	if err != nil {
		r.Log.Errorf("commande %s : %v", strings.Join(argv, " "), err)
		return err
	}
	r.Log.Infof("commande %s : ok", strings.Join(argv, " "))
	return nil
}

// copyFile writes through a temporary file then renames it, so a crash never
// leaves a half-written destination. With backup, the previous destination
// is kept as <to>.bak.
func copyFile(from, to string, backup bool) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s est un dossier", from)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(to), "."+filepath.Base(to)+".pm-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(tmp.Name(), info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	if backup {
		if old, err := os.ReadFile(to); err == nil {
			if err := writeAtomic(to+".bak", old, time.Now()); err != nil {
				return fmt.Errorf("sauvegarde .bak : %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("sauvegarde .bak : %w", err)
		}
	}
	return os.Rename(tmp.Name(), to)
}

// writeAtomic writes a new file next to path and renames it over path: an
// existing path (a link included) is replaced, never written through.
func writeAtomic(path string, data []byte, mod time.Time) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".pm-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(tmp.Name(), mod, mod); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
