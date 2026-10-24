package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidp57/process-monitor/internal/logx"
	"github.com/davidp57/process-monitor/internal/shortcut"
)

// A copy of the test binary plays the game. Next to it, <exe>.helper holds
// lines FILE=CONTENT: the "game" writes each file (deletes it for an empty
// CONTENT), then exits. The test process itself has no such file.
func TestMain(m *testing.M) {
	exe, _ := os.Executable()
	data, err := os.ReadFile(exe + ".helper")
	if err != nil {
		os.Exit(m.Run())
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		f, content, _ := strings.Cut(line, "=")
		if f == "sleep" { // a game that stays open
			ms, _ := strconv.Atoi(content)
			time.Sleep(time.Duration(ms) * time.Millisecond)
			continue
		}
		if content == "" {
			os.Remove(f)
			continue
		}
		os.MkdirAll(filepath.Dir(f), 0o755)
		os.WriteFile(f, []byte(content), 0o644)
	}
	os.Exit(0)
}

var seq int

// game copies the test binary under a unique name, with what it will do.
func game(t *testing.T, does map[string]string) string {
	t.Helper()
	seq++
	dst := filepath.Join(t.TempDir(), fmt.Sprintf("pm-wizard-game-%d-%d.exe", os.Getpid(), seq))
	self, _ := os.Executable()
	src, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(out, src)
	out.Close()
	var lines []string
	for f, c := range does {
		lines = append(lines, f+"="+c)
	}
	put(t, dst+".helper", strings.Join(lines, "\n"))
	return dst
}

func logger(t *testing.T) *logx.Logger {
	l, err := logx.New("", "", 0, func(logx.Line) {})
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func realEnv() Env {
	e := fakeEnv()
	e.Exists = func(p string) bool { _, err := os.Stat(p); return err == nil }
	return e
}

// A whole generic-game run, on real files: DEV changes the options and
// creates a file, VR deletes one; the original setup is back at the end.
func TestLearnAndApply(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "Settings")
	opts, extra, add := filepath.Join(root, "options.ini"), filepath.Join(root, "extra.ini"), filepath.Join(root, "Input", "pad.ini")
	put(t, opts, "original")
	put(t, extra, "extra")
	cfg := filepath.Join(d, "pm", "game.yaml")

	devGame := game(t, map[string]string{opts: "dev options", add: "pad"})
	w := New(realEnv(), cfg)
	r := &Runner{W: w, Log: logger(t), PM: `C:\pm\pm.exe`}
	run := func() any {
		t.Helper()
		s := w.Next()
		res, err := r.Do(context.Background(), *s.Action)
		if err != nil {
			t.Fatalf("%s : %v", s.ID, err)
		}
		w.Done(res)
		return res
	}
	dr := drive{t, w}
	dr.answer("case", CaseGame)
	dr.answer("launcher", devGame)
	dr.answer("track", "")
	dr.answer("roots", root)
	dr.answer("name", "game")
	dr.answer("config", "")
	dr.answer("profiles", "DEV", "VR")
	run() // save_original
	dr.answer("current", "")
	dr.answer("seed:DEV", "")
	run() // learn DEV
	if read(t, opts) != "dev options" {
		t.Fatalf("le jeu n'a pas tourné : %q", read(t, opts))
	}
	dr.answer("review:DEV", w.Next().Question.Default...)
	// VR's game is another program: point the launcher at it for the next round.
	vrGame := game(t, map[string]string{extra: ""})
	w.answers["launcher"] = []string{vrGame}
	dr.answer("seed:VR", "")
	run() // learn VR: starts from the original, not from DEV's setup
	if read(t, opts) != "original" || read(t, add) != "<absent>" || read(t, extra) != "<absent>" {
		t.Errorf("VR n'est pas parti de l'original : options %q, pad %q", read(t, opts), read(t, add))
	}
	dr.answer("review:VR", w.Next().Question.Default...)
	run() // restore_end
	dr.answer("icon:DEV", "")
	dr.answer("icon:VR", "")
	if read(t, opts) != "original" || read(t, extra) != "extra" || read(t, add) != "<absent>" {
		t.Errorf("original pas remis : options %q, extra %q, pad %q", read(t, opts), read(t, extra), read(t, add))
	}
	dr.answer("shortcuts", "${config_dir}")
	dr.answer("confirm", "yes")
	links := run().([]shortcut.Link)

	prof := filepath.Dir(cfg)
	for f, want := range map[string]string{
		`DEV\Settings\options.ini`: "dev options", `DEV\Settings\Input\pad.ini`: "pad", `DEV\Settings\extra.ini`: "extra",
		`VR\Settings\options.ini`: "original", `VR\Settings\Input\pad.ini`: "<absent>", `VR\Settings\extra.ini`: "<absent>",
	} {
		if got := read(t, filepath.Join(prof, f)); got != want {
			t.Errorf("%s = %q, voulu %q", f, got, want)
		}
	}
	if _, err := os.Stat(w.WizardDir()); err == nil {
		t.Error("dossier de l'assistant laissé")
	}
	if len(links) != 2 || links[0].Path != filepath.Join(prof, "game - DEV.lnk") {
		t.Errorf("raccourcis : %+v", links)
	}
	if _, err := r.Do(context.Background(), Action{Kind: ActApply}); err == nil || !strings.Contains(err.Error(), "existe déjà") {
		t.Errorf("config écrasée : %v", err)
	}
}

// An interrupted wizard left the original files saved: the next pm new on
// the same config puts them back before anything else.
func TestRecoverAfterInterruption(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "Settings")
	opts := filepath.Join(root, "options.ini")
	put(t, opts, "original")
	cfg := filepath.Join(d, "pm", "game.yaml")
	wiz := filepath.Join(filepath.Dir(cfg), ".pm-new")
	if _, err := SaveOriginal(wiz, []string{root}, nil); err != nil {
		t.Fatal(err)
	}
	put(t, opts, "left by a profile")

	w := New(realEnv(), cfg)
	dr := drive{t, w}
	dr.answer("case", CaseGame)
	dr.answer("launcher", game(t, nil))
	dr.answer("track", "")
	dr.answer("roots", root)
	dr.answer("name", "")
	dr.answer("config", "")
	s := w.Next()
	if s.ID != "recover" {
		t.Fatalf("étape %q, voulu recover", s.ID)
	}
	if _, err := (&Runner{W: w}).Do(context.Background(), *s.Action); err != nil {
		t.Fatal(err)
	}
	if read(t, opts) != "original" || HasOriginal(wiz) {
		t.Errorf("options %q, copie gardée %v", read(t, opts), HasOriginal(wiz))
	}
}

// The game left open (by a crash, a closed window) is found before any file
// is put back, and stopped only when the player asks.
func TestRunningGameIsFoundAndStopped(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "Settings")
	put(t, filepath.Join(root, "options.ini"), "original")
	exe := game(t, map[string]string{"sleep": "20000"})
	open := exec.Command(exe)
	if err := open.Start(); err != nil {
		t.Fatal(err)
	}
	defer open.Process.Kill()

	w := New(realEnv(), filepath.Join(d, "pm", "game.yaml"))
	r := &Runner{W: w, Log: logger(t)}
	dr := drive{t, w}
	dr.answer("case", CaseGame)
	dr.answer("launcher", exe)
	dr.answer("track", "")
	dr.answer("roots", root)
	dr.answer("name", "game")
	dr.answer("config", "")
	dr.answer("profiles", "DEV")
	res, err := r.Do(context.Background(), *w.Next().Action)
	if err != nil {
		t.Fatal(err)
	}
	w.Done(res)
	dr.answer("current", "")
	dr.answer("seed:DEV", "")
	_, err = r.Do(context.Background(), *w.Next().Action)
	var running *Running
	if !errors.As(err, &running) || len(running.PIDs) != 1 || running.PIDs[0] != open.Process.Pid {
		t.Fatalf("jeu ouvert non signalé : %v", err)
	}
	if err := r.Abandon(); err == nil || !HasOriginal(w.WizardDir()) {
		t.Errorf("abandon sous un jeu ouvert : %v, copie gardée %v", err, HasOriginal(w.WizardDir()))
	}
	if err := r.StopGame(running); err != nil {
		t.Fatal(err)
	}
	if err := r.checkGame(); err != nil {
		t.Errorf("toujours là : %v", err)
	}
}

// Each way to start from existing files puts them in the game before it
// starts, and the report then shows them; files outside the watched
// folders are left out.
func TestSeedBeforePlaying(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "Settings")
	opts, pad := filepath.Join(root, "options.ini"), filepath.Join(root, "Input", "pad.ini")
	put(t, opts, "original")
	// a: a folder laid out like the game; Other is not a watched folder.
	folder := filepath.Join(d, "backup")
	put(t, filepath.Join(folder, "Settings", "options.ini"), "from folder")
	put(t, filepath.Join(folder, "Other", "x.ini"), "outside")
	// b: a loose file.
	loose := filepath.Join(d, "loose", "pad.ini")
	put(t, loose, "loose pad")
	// c: an older pm config.
	old := filepath.Join(d, "old", "pm-game.yaml")
	put(t, filepath.Join(d, "old", "VR", "options.ini"), "from old config")
	put(t, old, "command: x.exe\nhooks:\n  sync:\n    - '"+`${config_dir}\${profile}\options.ini`+" <-> "+opts+"'\nprofiles: { VR: {} }\n")

	w := New(realEnv(), filepath.Join(d, "pm", "game.yaml"))
	w.env.Profiles = func(p string) ([]string, error) { return []string{"VR"}, nil }
	r := &Runner{W: w, Log: logger(t)}
	dr := drive{t, w}
	run := func() map[string]string {
		t.Helper()
		res, err := r.Do(context.Background(), *w.Next().Action)
		if err != nil {
			t.Fatal(err)
		}
		w.Done(res)
		m, _ := res.(map[string]string)
		return m
	}
	dr.answer("case", CaseGame)
	dr.answer("launcher", game(t, nil))
	dr.answer("track", "")
	dr.answer("roots", root)
	dr.answer("name", "game")
	dr.answer("config", "")
	dr.answer("profiles", "A", "B", "C")
	run() // save_original
	dr.answer("current", "")

	dr.answer("seed:A", seedFolder)
	dr.answer("seed_dir:A", folder)
	if got := run(); len(got) != 1 || got[opts] == "" || read(t, opts) != "from folder" {
		t.Errorf("A : %v, options %q", got, read(t, opts))
	}
	dr.answer("review:A", w.Next().Question.Default...)

	dr.answer("seed:B", seedFiles)
	dr.answer("seed_files:B", loose)
	dr.answer(seedTo("B", 0), pad)
	if got := run(); len(got) != 1 || got[pad] == "" || read(t, opts) != "original" {
		t.Errorf("B : %v, options %q (l'original devait être remis)", got, read(t, opts))
	}
	dr.answer("review:B", w.Next().Question.Default...)

	dr.answer("seed:C", seedConfig)
	dr.answer("seed_config:C", old)
	dr.answer("seed_profile:C", "")
	if got := run(); len(got) != 1 || read(t, opts) != "from old config" {
		t.Errorf("C : %v, options %q", got, read(t, opts))
	}
}

// An older config moved into a backup still names its old place: its
// files are looked for next to it first, the longest matching end of path
// winning, and the path as written only when nothing is found there.
func TestUnderConfigFindsAMovedBackup(t *testing.T) {
	d := t.TempDir()
	backup := filepath.Join(d, "Profiles-2026.10.24", "Profiles")
	put(t, filepath.Join(backup, "STD", "options.lua"), "std")
	put(t, filepath.Join(backup, "DEV", "options.lua"), "dev")
	gone := filepath.Join(d, "Saved Games", "DCS", "Profiles", "STD", "options.lua")
	if got := underConfig(gone, backup); got != filepath.Join(backup, "STD", "options.lua") {
		t.Errorf("trouvé %q", got)
	}
	elsewhere := filepath.Join(d, "other", "x.lua")
	if got := underConfig(elsewhere, backup); got != elsewhere {
		t.Errorf("rien à côté : %q", got)
	}
}
