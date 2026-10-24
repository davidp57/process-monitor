package shortcut

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidp57/process-monitor/internal/config"
)

func put(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteAndReadBack(t *testing.T) {
	d := t.TempDir()
	l := Link{Path: filepath.Join(d, "sub", "DCS - VR.lnk"), Target: `C:\Windows\System32\cmd.exe`,
		Args: `run "C:\x\dcs.yaml" --profile VR`, Workdir: `C:\x`, Icon: `C:\Windows\System32\shell32.dll`, Minimized: true}
	kept, err := Write(l)
	if err != nil {
		t.Fatal(err)
	}
	if kept != "" {
		t.Errorf("kept = %q, rien n'existait", kept)
	}
	got, err := Read(l.Path)
	if err != nil {
		t.Fatal(err)
	}
	if got != l {
		t.Errorf("relu %+v\nvoulu %+v", got, l)
	}
}

func TestWriteKeepsOld(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "a.lnk")
	put(t, p, "old shortcut")
	put(t, p+".old", "older")
	kept, err := Write(Link{Path: p, Target: `C:\Windows\System32\cmd.exe`})
	if err != nil {
		t.Fatal(err)
	}
	if kept != p+".old" {
		t.Errorf("kept = %q", kept)
	}
	if b, _ := os.ReadFile(p + ".old"); string(b) != "old shortcut" {
		t.Errorf(".old = %q", b)
	}
	if got, err := Read(p); err != nil || got.Target != `C:\Windows\System32\cmd.exe` {
		t.Errorf("relu %+v, %v", got, err)
	}
}

func TestForConfig(t *testing.T) {
	d := t.TempDir()
	cfg := filepath.Join(d, "dcs.yaml")
	put(t, filepath.Join(d, "dev.ico"), "")
	put(t, cfg, `name: dcs
command: C:\Windows\System32\cmd.exe
ui: tray
icon: game.png
shortcuts: { mode: run, folder: links }
profiles:
  DEV: { icon: dev.ico }
  VR: {}
`)
	links, err := ForConfig(cfg, "", `C:\pm\pm.exe`)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 {
		t.Fatalf("links = %+v", links)
	}
	dev, vr := links[0], links[1]
	want := Link{Path: filepath.Join(d, "links", "dcs - DEV.lnk"), Target: `C:\pm\pm.exe`,
		Args: `run "` + cfg + `" --profile DEV`, Workdir: d, Icon: filepath.Join(d, "dev.ico"), Minimized: true}
	if dev != want {
		t.Errorf("DEV %+v\nvoulu %+v", dev, want)
	}
	// A .png is fine in the notification area, not on a shortcut: the program's icon then.
	if vr.Icon != `C:\Windows\System32\cmd.exe` || !strings.HasSuffix(vr.Args, "--profile VR") {
		t.Errorf("VR %+v", vr)
	}

	only, err := ForConfig(cfg, "VR", `C:\pm\pm.exe`)
	if err != nil || len(only) != 1 || only[0] != vr {
		t.Errorf("--profile VR : %+v, %v", only, err)
	}
}

func TestForConfigWithoutProfilesOrSection(t *testing.T) {
	d := t.TempDir()
	cfg := filepath.Join(d, "sim.yaml")
	put(t, cfg, "command: C:\\Windows\\System32\\cmd.exe\nshortcuts: { mode: watch }\n")
	links, err := ForConfig(cfg, "", `C:\pm\pm.exe`)
	if err != nil || len(links) != 1 {
		t.Fatalf("%+v, %v", links, err)
	}
	if l := links[0]; l.Path != filepath.Join(d, "sim.lnk") || l.Args != `watch "`+cfg+`"` || l.Minimized {
		t.Errorf("%+v", l)
	}

	put(t, cfg, "command: a.exe\n")
	if _, err := ForConfig(cfg, "", `C:\pm\pm.exe`); err == nil || !strings.Contains(err.Error(), "shortcuts") {
		t.Errorf("sans section : %v", err)
	}
}

// From the menu, a config without the section gets one added at its end:
// the comments and the rest stay as they were.
func TestAddSection(t *testing.T) {
	d := t.TempDir()
	cfg := filepath.Join(d, "pm-dcs.yaml")
	orig := "# DCS par profil\nname: dcs\ncommand: " + `C:\Windows\System32\cmd.exe` + "\nprofiles: { DEV: {}, VR: {} }" // no final newline
	put(t, cfg, orig)
	if HasSection(cfg) {
		t.Fatal("section vue alors qu'il n'y en a pas")
	}
	if err := AddSection(cfg, "run", []string{"${desktop}", "${start_menu}"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(cfg)
	if !strings.HasPrefix(string(b), orig+"\n") || !HasSection(cfg) {
		t.Errorf("fichier :\n%s", b)
	}
	links, err := ForConfig(cfg, "", `C:\pm\pm.exe`)
	// Two profiles, two folders: four shortcuts.
	if err != nil || len(links) != 4 || filepath.Base(links[0].Path) != "dcs - DEV.lnk" || filepath.Dir(links[0].Path) == filepath.Dir(links[1].Path) {
		t.Errorf("%+v, %v", links, err)
	}

	// A section pm refuses: the file is put back as it was.
	cfg2 := filepath.Join(d, "other.yaml")
	put(t, cfg2, orig)
	if err := AddSection(cfg2, "start", []string{"${desktop}"}); err == nil {
		t.Error("mode invalide accepté")
	}
	if b, _ := os.ReadFile(cfg2); string(b) != orig {
		t.Errorf("pas remis :\n%s", b)
	}
}

// An icon goes into the profile that has none, the file otherwise as it was.
func TestSetIcon(t *testing.T) {
	d := t.TempDir()
	ico := filepath.Join(d, "Icons", "dcs-dev.ico")
	put(t, ico, "")
	cfg := filepath.Join(d, "pm-dcs.yaml")
	head := "# DCS\nname: dcs\ncommand: " + `C:\Windows\System32\cmd.exe` + "\nprofiles:\n"
	put(t, cfg, head+"  DEV: {}   \n  'VR':\n    # the VR one\n    hooks: { before: [] }\n  STD: { hooks: {} }\n")
	if err := SetIcon(cfg, "DEV", ico); err != nil {
		t.Fatal(err)
	}
	if err := SetIcon(cfg, "VR", ico); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(cfg)
	want := head + "  DEV:\n    icon: '" + ico + "'\n  'VR':\n    icon: '" + ico + "'\n    # the VR one\n    hooks: { before: [] }\n  STD: { hooks: {} }\n"
	if string(b) != want {
		t.Errorf("fichier :\n%s\nvoulu :\n%s", b, want)
	}
	// Inline with content: left to the player, the file untouched.
	if err := SetIcon(cfg, "STD", ico); err == nil {
		t.Error("profil en ligne modifié")
	}
	if b2, _ := os.ReadFile(cfg); string(b2) != want {
		t.Error("fichier changé malgré l'échec")
	}

	// Without profiles: at the end.
	cfg2 := filepath.Join(d, "sim.yaml")
	put(t, cfg2, "command: "+`C:\Windows\System32\cmd.exe`+"\n\n")
	if err := SetIcon(cfg2, "", ico); err != nil {
		t.Fatal(err)
	}
	if c, err := config.Load(cfg2, ""); err != nil || c.Icon != ico {
		t.Errorf("%+v, %v", c, err)
	}
}

// A config that does not say how it shows itself gets the ui key at its end.
func TestSetUI(t *testing.T) {
	d := t.TempDir()
	cfg := filepath.Join(d, "pm-dcs.yaml")
	orig := "# DCS\ncommand: " + `C:\Windows\System32\cmd.exe` + "\nprofiles:\n  DEV: {}\n\n"
	put(t, cfg, orig)
	if HasUI(cfg) {
		t.Fatal("ui vu alors qu'il n'y est pas")
	}
	if err := SetUI(cfg, "tray"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(cfg)
	if string(b) != "# DCS\ncommand: "+`C:\Windows\System32\cmd.exe`+"\nprofiles:\n  DEV: {}\nui: tray\n\n" || !HasUI(cfg) {
		t.Errorf("fichier :\n%s", b)
	}
	// The shortcut then opens minimized.
	put(t, cfg, string(b)+"shortcuts: { mode: run }\n")
	if links, err := ForConfig(cfg, "", `C:\pm\pm.exe`); err != nil || !links[0].Minimized {
		t.Errorf("%+v, %v", links, err)
	}
	cfg2 := filepath.Join(d, "bad.yaml")
	put(t, cfg2, orig)
	if err := SetUI(cfg2, "window"); err == nil {
		t.Error("ui invalide accepté")
	}
	if b, _ := os.ReadFile(cfg2); string(b) != orig {
		t.Errorf("pas remis :\n%s", b)
	}
}
