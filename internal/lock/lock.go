// Package lock holds a per-session lock so two sessions on the same program
// (two DCS profiles, say) can never overlap. The operating system releases
// it when pm exits, even on a crash, so a stale file never blocks.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Lock is a held session lock.
type Lock struct {
	f    *os.File
	Path string
}

// Busy is returned when another pm already holds the lock.
type Busy struct {
	Path  string
	Owner string // what the holder wrote, may be empty
}

func (b *Busy) Error() string {
	if b.Owner == "" {
		return i18n.T("lock.busy")
	}
	return i18n.T("lock.busy_owner", b.Owner)
}

// Dir is where locks live.
func Dir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pm", "locks"), nil
}

// Acquire takes the lock called name and records owner in it.
func Acquire(name, owner string) (*Lock, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, sanitize(name)+".lock")
	f, err := open(path)
	if err != nil {
		if isBusy(err) {
			b, _ := os.ReadFile(path)
			return nil, &Busy{Path: path, Owner: strings.TrimSpace(string(b))}
		}
		return nil, err
	}
	_ = f.Truncate(0)
	fmt.Fprintf(f, "%s (pid %d, depuis %s)\n", owner, os.Getpid(), time.Now().Format("15:04:05"))
	_ = f.Sync()
	return &Lock{f: f, Path: path}, nil
}

// Release frees the lock.
func (l *Lock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	_ = l.f.Truncate(0)
	err := l.f.Close()
	l.f = nil
	return err
}

func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || r < 32 {
			return '_'
		}
		return r
	}, name)
}
