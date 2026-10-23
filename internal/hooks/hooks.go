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

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/logx"
)

// Runner executes hooks and logs each step.
type Runner struct {
	Log    *logx.Logger
	Backup bool
}

// Before copies every sync pair in (profile -> live), then runs the before
// actions. It stops at the first failure: launching with half a profile in
// place is worse than not launching.
func (r *Runner) Before(ctx context.Context, c *config.Config) error {
	for _, p := range c.Sync {
		if err := r.copyFile(p.Source, p.Target); err != nil {
			return err
		}
	}
	for _, a := range c.Hooks.Before {
		if err := r.run(ctx, a); err != nil {
			return err
		}
	}
	return nil
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
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
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
		if err := copyPlain(to, to+".bak"); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("sauvegarde .bak : %w", err)
		}
	}
	return os.Rename(tmp.Name(), to)
}

func copyPlain(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}
