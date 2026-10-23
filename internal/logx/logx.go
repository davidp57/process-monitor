// Package logx writes session events to a timestamped file and forwards each
// line to a sink (console or TUI).
package logx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/davidp57/process-monitor/internal/i18n"
)

type Level string

const (
	Info  Level = "INFO"
	Warn  Level = "WARN"
	Error Level = "ERR "
	Out   Level = "OUT "
)

// Line is one log entry.
type Line struct {
	Time  time.Time
	Level Level
	Text  string
}

func (l Line) String() string {
	return fmt.Sprintf("%s %s %s", l.Time.Format("15:04:05"), l.Level, l.Text)
}

// Logger is safe for concurrent use.
type Logger struct {
	mu     sync.Mutex
	file   io.WriteCloser
	sink   func(Line)
	errors int
	Path   string
}

// New opens <dir>/<prefix>-<timestamp>.log and keeps only the `keep` most
// recent files with that prefix. An empty dir logs to the sink only.
func New(dir, prefix string, keep int, sink func(Line)) (*Logger, error) {
	l := &Logger{sink: sink}
	if dir == "" {
		return l, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	l.Path = filepath.Join(dir, fmt.Sprintf("%s-%s.log", prefix, time.Now().Format("20060102-150405")))
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	l.file = f
	if dropped, err := prune(dir, prefix, keep); err != nil {
		l.Warn(i18n.T("log.prune_failed", err))
	} else {
		for _, p := range dropped {
			l.Info(i18n.T("log.pruned", filepath.Base(p)))
		}
	}
	return l, nil
}

// SetSink replaces the line consumer.
func (l *Logger) SetSink(sink func(Line)) {
	l.mu.Lock()
	l.sink = sink
	l.mu.Unlock()
}

func (l *Logger) log(level Level, format string, args ...any) {
	line := Line{Time: time.Now(), Level: level, Text: fmt.Sprintf(format, args...)}
	l.mu.Lock()
	if level == Error {
		l.errors++
	}
	if l.file != nil {
		fmt.Fprintf(l.file, "%s %s %s\n", line.Time.Format("2006-01-02 15:04:05.000"), line.Level, line.Text)
	}
	sink := l.sink
	l.mu.Unlock()
	// Outside the lock: the TUI sink blocks until the interface takes the
	// line, and the interface itself reads Errors().
	if sink != nil {
		sink(line)
	}
}

// Info, Warn and Error log a message already formatted (from the catalog).
func (l *Logger) Info(msg string)  { l.log(Info, "%s", msg) }
func (l *Logger) Warn(msg string)  { l.log(Warn, "%s", msg) }
func (l *Logger) Error(msg string) { l.log(Error, "%s", msg) }

func (l *Logger) Infof(format string, args ...any)  { l.log(Info, format, args...) }
func (l *Logger) Warnf(format string, args ...any)  { l.log(Warn, format, args...) }
func (l *Logger) Errorf(format string, args ...any) { l.log(Error, format, args...) }

// Output records one line printed by the supervised program. Control
// characters are dropped: escape sequences from the program must not drive
// the terminal pm shows its own state in.
func (l *Logger) Output(text string) { l.log(Out, "%s", printable(text)) }

func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || (r >= 0x20 && r != 0x7f && !(r >= 0x80 && r < 0xa0)) {
			return r
		}
		return -1
	}, s)
}

// Errors returns how many errors were logged.
func (l *Logger) Errors() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.errors
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func prune(dir, prefix string, keep int) ([]string, error) {
	if keep <= 0 {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var logs []string
	for _, e := range entries {
		n := e.Name()
		// prefix-YYYYMMDD-HHMMSS.log: the timestamp keeps "dcs-DEV" from matching "dcs-DEV2".
		rest, ok := strings.CutPrefix(n, prefix+"-")
		if ok && !e.IsDir() && strings.HasSuffix(n, ".log") && len(rest) == len("20060102-150405.log") {
			logs = append(logs, filepath.Join(dir, n))
		}
	}
	if len(logs) <= keep {
		return nil, nil
	}
	sort.Strings(logs) // timestamped names sort chronologically
	var dropped []string
	for _, p := range logs[:len(logs)-keep] {
		if err := os.Remove(p); err != nil {
			return dropped, err
		}
		dropped = append(dropped, p)
	}
	return dropped, nil
}
