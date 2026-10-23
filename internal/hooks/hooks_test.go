package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/logx"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func put(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runner(t *testing.T) *Runner {
	l, err := logx.New("", "", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Runner{Log: l, Backup: true}
}

func TestSyncRoundTrip(t *testing.T) {
	d := t.TempDir()
	prof, live := filepath.Join(d, "prof", "options.lua"), filepath.Join(d, "live", "config", "options.lua")
	put(t, prof, "profile")
	put(t, live, "previous")
	c := &config.Config{Sync: []config.SyncPair{{Source: prof, Target: live}}}
	r := runner(t)

	if _, err := r.Before(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if read(t, live) != "profile" || read(t, live+".bak") != "previous" {
		t.Errorf("before : live=%q bak=%q", read(t, live), read(t, live+".bak"))
	}

	put(t, live, "changed in game")
	if err := r.After(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if read(t, prof) != "changed in game" || read(t, prof+".bak") != "profile" {
		t.Errorf("after : prof=%q bak=%q", read(t, prof), read(t, prof+".bak"))
	}
	entries, _ := os.ReadDir(filepath.Dir(live))
	for _, e := range entries {
		if e.Name() != "options.lua" && e.Name() != "options.lua.bak" {
			t.Errorf("fichier temporaire oublié : %s", e.Name())
		}
	}
}

func TestBeforeStopsAtFirstFailure(t *testing.T) {
	d := t.TempDir()
	marker := filepath.Join(d, "marker")
	put(t, marker, "x")
	c := &config.Config{
		Sync:  []config.SyncPair{{Source: filepath.Join(d, "missing"), Target: filepath.Join(d, "t")}},
		Hooks: config.Hooks{Before: []config.Action{{Delete: marker}}},
	}
	r := runner(t)
	if _, err := r.Before(context.Background(), c); err == nil {
		t.Fatal("pas d'erreur sur source absente")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("l'action suivante a été exécutée malgré l'échec")
	}
	if r.Log.Errors() != 1 {
		t.Errorf("erreurs journalisées = %d", r.Log.Errors())
	}
}

func TestBeforeFailureRestoresLiveFiles(t *testing.T) {
	d := t.TempDir()
	p1, l1 := filepath.Join(d, "p1"), filepath.Join(d, "live", "l1")
	p2, l2 := filepath.Join(d, "p2"), filepath.Join(d, "live", "l2")
	put(t, p1, "profile 1")
	put(t, l1, "live 1")
	put(t, p2, "profile 2") // l2 does not exist yet
	c := &config.Config{
		Sync:  []config.SyncPair{{Source: p1, Target: l1}, {Source: p2, Target: l2}},
		Hooks: config.Hooks{Before: []config.Action{{Copy: &config.Copy{From: filepath.Join(d, "missing"), To: filepath.Join(d, "x")}}}},
	}
	if _, err := runner(t).Before(context.Background(), c); err == nil {
		t.Fatal("pas d'erreur")
	}
	if read(t, l1) != "live 1" {
		t.Errorf("l1 = %q, pas remis en état", read(t, l1))
	}
	if _, err := os.Stat(l2); !os.IsNotExist(err) {
		t.Error("l2 aurait dû disparaître : il n'existait pas avant")
	}
}

// A command that leaves a program in the background must not hold pm.
// A command that leaves a program in the background must not hold pm: exec
// returns while that program still runs.
func TestExecDoesNotWaitForBackgroundChild(t *testing.T) {
	mark := filepath.Join(t.TempDir(), "child-exited")
	t.Setenv("PM_HELPER", "spawn:"+mark)
	var lines []string
	l, _ := logx.New("", "", 0, func(line logx.Line) { lines = append(lines, line.Text) })
	r := &Runner{Log: l}
	begin := time.Now()
	if err := r.exec(context.Background(), []string{self(t)}); err != nil {
		t.Fatal(err)
	}
	took := time.Since(begin)
	if _, err := os.Stat(mark); err == nil {
		t.Fatal("le programme en arrière-plan avait déjà fini : le test ne prouve rien")
	}
	if took > 7*time.Second {
		t.Errorf("exec a attendu %s", took)
	}
	for _, line := range lines { // stop the background child by the PID it printed
		if pid, ok := strings.CutPrefix(line, "child "); ok {
			if n, err := strconv.Atoi(pid); err == nil {
				if p, err := os.FindProcess(n); err == nil {
					p.Kill()
				}
			}
		}
	}
}

func TestExecTimeout(t *testing.T) {
	t.Setenv("PM_HELPER", "sleep:10000:"+filepath.Join(t.TempDir(), "never"))
	r := runner(t)
	r.Timeout = 500 * time.Millisecond
	begin := time.Now()
	err := r.exec(context.Background(), []string{self(t)})
	if err == nil || !strings.Contains(err.Error(), "délai") || time.Since(begin) > 3*time.Second {
		t.Errorf("err = %v après %s", err, time.Since(begin))
	}
}

// Relative arguments of an exec action are taken from the config's folder.
func TestExecRunsInRunnerDir(t *testing.T) {
	d := t.TempDir()
	t.Setenv("PM_HELPER", "touch:here.txt")
	r := runner(t)
	r.Dir = d
	if err := r.exec(context.Background(), []string{self(t)}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, "here.txt")); err != nil {
		t.Error("le fichier relatif n'a pas été créé dans Dir")
	}
}

// Replaying the before actions must not overwrite the .bak of the original.
func TestBeforeRestartKeepsOriginalBackup(t *testing.T) {
	d := t.TempDir()
	from, to := filepath.Join(d, "profile"), filepath.Join(d, "live")
	put(t, from, "profile")
	put(t, to, "original")
	c := &config.Config{Hooks: config.Hooks{Before: []config.Action{{Copy: &config.Copy{From: from, To: to}}}}}
	r := runner(t)
	if _, err := r.Before(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	put(t, to, "changed by the program")
	if err := r.BeforeRestart(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if got := read(t, to+".bak"); got != "original" {
		t.Errorf(".bak = %q, l'original est perdu", got)
	}
}

// Undo puts the live files back when the session is cancelled before launch.
func TestBeforeUndo(t *testing.T) {
	d := t.TempDir()
	prof, live := filepath.Join(d, "p"), filepath.Join(d, "l")
	put(t, prof, "profile")
	put(t, live, "before")
	c := &config.Config{Sync: []config.SyncPair{{Source: prof, Target: live}}}
	undo, err := runner(t).Before(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	undo()
	if read(t, live) != "before" {
		t.Errorf("live = %q", read(t, live))
	}
}

func self(t *testing.T) string {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe
}

// The test binary doubles as the program an exec action runs, driven by PM_HELPER:
//
//	spawn:MARK        start a copy of itself in sleep mode, print "child <pid>", exit at once
//	sleep:MS:MARK     sleep, then write MARK
//	touch:NAME        create NAME in the current directory
func TestMain(m *testing.M) {
	mode, arg, _ := strings.Cut(os.Getenv("PM_HELPER"), ":")
	switch mode {
	case "":
		os.Exit(m.Run())
	case "spawn":
		self, _ := os.Executable()
		child := exec.Command(self)
		child.Env = append(os.Environ(), "PM_HELPER=sleep:8000:"+arg)
		child.Stdout, child.Stderr = os.Stdout, os.Stderr // holds the output pipe open
		if err := child.Start(); err != nil {
			os.Exit(90)
		}
		fmt.Printf("child %d\n", child.Process.Pid)
		os.Exit(0)
	case "sleep":
		ms, mark, _ := strings.Cut(arg, ":")
		n, _ := strconv.Atoi(ms)
		time.Sleep(time.Duration(n) * time.Millisecond)
		os.WriteFile(mark, nil, 0o644)
		os.Exit(0)
	case "touch":
		os.WriteFile(arg, nil, 0o644)
		os.Exit(0)
	}
	os.Exit(98)
}

func TestAfterRunsEverything(t *testing.T) {
	d := t.TempDir()
	dir := filepath.Join(d, "cache")
	put(t, filepath.Join(dir, "a.txt"), "x")
	c := &config.Config{
		Sync:  []config.SyncPair{{Source: filepath.Join(d, "p"), Target: filepath.Join(d, "missing")}},
		Hooks: config.Hooks{After: []config.Action{{Delete: dir}, {Delete: filepath.Join(d, "absent")}}},
	}
	if err := runner(t).After(context.Background(), c); err == nil {
		t.Fatal("l'échec de la copie retour n'est pas remonté")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("le dossier n'a pas été supprimé après l'échec de la copie")
	}
}

func TestBackupDisabled(t *testing.T) {
	d := t.TempDir()
	from, to := filepath.Join(d, "a"), filepath.Join(d, "b")
	put(t, from, "new")
	put(t, to, "old")
	r := runner(t)
	r.Backup = false
	if err := r.run(context.Background(), config.Action{Copy: &config.Copy{From: from, To: to}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(to + ".bak"); !os.IsNotExist(err) {
		t.Error(".bak créé alors que backup est désactivé")
	}
}
