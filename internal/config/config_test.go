package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "game.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const profiled = `
vars:
  saved: C:\Saved
  prof: ${saved}\Profiles\${profile}
command: C:\game\updater.exe
workdir: ${prof}
track: game.exe
ready: { file: "${saved}\\game.lock" }
hooks:
  sync:
    - ${prof}\options.lua <-> ${saved}\config\options.lua
profiles:
  DEV:
    hooks:
      sync: [ "${prof}\\joy.lua <-> ${saved}\\Input\\joy.lua" ]
  VR: {}
`

func TestProfileExpandsAndAppends(t *testing.T) {
	c, err := Load(write(t, profiled), "DEV")
	if err != nil {
		t.Fatal(err)
	}
	if c.Workdir != `C:\Saved\Profiles\DEV` {
		t.Errorf("workdir = %q", c.Workdir)
	}
	want := []SyncPair{
		{`C:\Saved\Profiles\DEV\options.lua`, `C:\Saved\config\options.lua`},
		{`C:\Saved\Profiles\DEV\joy.lua`, `C:\Saved\Input\joy.lua`},
	}
	if len(c.Sync) != len(want) {
		t.Fatalf("sync = %v", c.Sync)
	}
	for i := range want {
		if c.Sync[i] != want[i] {
			t.Errorf("sync[%d] = %v, want %v", i, c.Sync[i], want[i])
		}
	}
	if c.Ready.File != `C:\Saved\game.lock` || c.Ready.Timeout.D() != 2*time.Minute {
		t.Errorf("ready = %+v", c.Ready)
	}
	if c.Name != "game" || c.ProcessName() != "game.exe" {
		t.Errorf("name = %q, process = %q", c.Name, c.ProcessName())
	}
}

func TestOtherProfileDoesNotGetDevSync(t *testing.T) {
	c, err := Load(write(t, profiled), "VR")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Sync) != 1 || !strings.Contains(c.Sync[0].Source, `\VR\`) {
		t.Errorf("sync = %v", c.Sync)
	}
}

func TestProfileRequiredWhenDefined(t *testing.T) {
	_, err := Load(write(t, profiled), "")
	if err == nil || !strings.Contains(err.Error(), "DEV, VR") {
		t.Errorf("err = %v", err)
	}
	_, err = Load(write(t, profiled), "STD")
	if err == nil || !strings.Contains(err.Error(), "inconnu") {
		t.Errorf("err = %v", err)
	}
}

func TestErrors(t *testing.T) {
	cases := map[string]string{
		"unknown key":   "command: a.exe\nkill_existing: true\n",
		"undefined var": "command: ${nope}\\a.exe\n",
		"var loop":      "vars: {a: '${b}', b: '${a}'}\ncommand: ${a}\n",
		"bad sync":      "command: a.exe\nhooks: {sync: [\"a -> b\"]}\n",
		"two actions":   "command: a.exe\nhooks:\n  before:\n    - {delete: x, exec: [y]}\n",
		"bad duration":  "command: a.exe\nwatch: {delay: 2}\n",
		"bad restart":   "command: a.exe\nwatch: {restart: sometimes}\n",
		"no command":    "track: a.exe\n",
		"profile asked": "command: a.exe\n",
	}
	for name, yml := range cases {
		profile := ""
		if name == "profile asked" {
			profile = "X"
		}
		if _, err := Load(write(t, yml), profile); err == nil {
			t.Errorf("%s : pas d'erreur", name)
		}
	}
}

// Each guard must fail for its own reason, not on a YAML typo.
func TestGuards(t *testing.T) {
	cases := map[string]struct{ yml, want string }{
		"track path":   {"command: a.exe\ntrack: C:\\x\\dcs.exe\n", "nom de process seul"},
		"track no ext": {"command: a.exe\ntrack: dcs\n", "nom de process seul"},
		"reserved var": {"vars: {profile: X}\ncommand: a.exe\n", "réservé"},
		// Without profiles, ${profile} must not silently become "" (C:\P\ would be deleted).
		"no profiles": {"command: a.exe\nhooks: {before: [{delete: 'C:\\P\\${profile}'}]}\n", "${profile} non définie"},
		"dup target":  {"command: a.exe\nhooks: {sync: ['a <-> C:\\x', 'b <-> c:\\X']}\n", "cible de deux lignes"},
		"dup source":  {"command: a.exe\nhooks: {sync: ['C:\\p <-> a', 'c:\\P <-> b']}\n", "source de deux lignes"},
		"two docs":    {"command: a.exe\n---\nhooks: {before: [{delete: x}]}\n", "un seul document"},
		"neg delay":   {"command: a.exe\nwatch: {delay: -2s}\n", "négative"},
		"kill path":   {"command: a.exe\nkill_on_exit: ['C:\\x\\srs.exe']\n", "kill_on_exit"},
	}
	for name, tc := range cases {
		_, err := Load(write(t, tc.yml), "")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s : err = %v, attendu %q", name, err, tc.want)
		}
	}
}

func TestNamesAndPrograms(t *testing.T) {
	p := write(t, "name: ${who}-game\nvars: {who: dav}\ncommand: tools\\app\nhooks: {before: [{exec: [bin\\x.cmd, arg]}, {exec: [cmd, /c, ok]}]}\n")
	c, err := Load(p, "")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(p)
	if c.Name != "dav-game" {
		t.Errorf("name = %q", c.Name)
	}
	if c.Command != filepath.Join(dir, "tools", "app") || c.LauncherName() != "app.exe" {
		t.Errorf("command = %q, launcher = %q", c.Command, c.LauncherName())
	}
	if c.Hooks.Before[0].Exec[0] != filepath.Join(dir, "bin", "x.cmd") || c.Hooks.Before[1].Exec[0] != "cmd" {
		t.Errorf("exec = %q / %q", c.Hooks.Before[0].Exec[0], c.Hooks.Before[1].Exec[0])
	}
}

// The documented examples must stay loadable.
func TestExamplesLoad(t *testing.T) {
	files, _ := filepath.Glob("../../examples/*.yaml")
	if len(files) == 0 {
		t.Fatal("aucun exemple trouvé")
	}
	for _, f := range files {
		c, err := Load(f, "")
		if err != nil && strings.Contains(err.Error(), "--profile") {
			raw, _ := Load(f, "DEV") // read the profile list from a first profile
			if raw == nil {
				t.Fatalf("%s : %v", f, err)
			}
			for _, p := range raw.ProfileNames() {
				if _, err := Load(f, p); err != nil {
					t.Errorf("%s / %s : %v", f, p, err)
				}
			}
			continue
		}
		if err != nil {
			t.Errorf("%s : %v", f, err)
		} else if c.Command == "" {
			t.Errorf("%s : command vide", f)
		}
	}
}

func TestEnvAndDefaults(t *testing.T) {
	t.Setenv("PM_TEST_DIR", `D:\x`)
	c, err := Load(write(t, "command: ${env:PM_TEST_DIR}\\a.exe\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Command != `D:\x\a.exe` {
		t.Errorf("command = %q", c.Command)
	}
	if c.IfRunning != IfRunningAbort || c.Watch.Restart != RestartAlways || !c.BackupEnabled() || c.Log.Keep != 5 {
		t.Errorf("defaults = %+v", c)
	}
	if c.Log.Dir != filepath.Join(filepath.Dir(c.Path), "logs") {
		t.Errorf("log dir = %q", c.Log.Dir)
	}
	if c.ProcessName() != "a.exe" {
		t.Errorf("process = %q", c.ProcessName())
	}
}
