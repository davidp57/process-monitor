package hooks

import (
	"context"
	"os"
	"path/filepath"
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

	if err := r.Before(context.Background(), c); err != nil {
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
	if err := r.Before(context.Background(), c); err == nil {
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
	if err := runner(t).Before(context.Background(), c); err == nil {
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
func TestExecDoesNotWaitForBackgroundChild(t *testing.T) {
	r := runner(t)
	begin := time.Now()
	err := r.exec(context.Background(), []string{"cmd", "/c", "start", "/b", "ping", "-n", "12", "127.0.0.1"}) // the background ping ends on its own
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(begin); d > 10*time.Second {
		t.Errorf("exec a attendu %s", d)
	}
}

func TestExecTimeout(t *testing.T) {
	r := runner(t)
	r.Timeout = 500 * time.Millisecond
	err := r.exec(context.Background(), []string{"ping", "-n", "10", "127.0.0.1"})
	if err == nil || !strings.Contains(err.Error(), "délai") {
		t.Errorf("err = %v", err)
	}
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
