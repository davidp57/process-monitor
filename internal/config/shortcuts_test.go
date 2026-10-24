package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestShortcutsDefaults(t *testing.T) {
	c, err := Load(write(t, "command: a.exe\nshortcuts: { mode: watch }\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Shortcuts.Name != "game" || len(c.Shortcuts.Folders) != 1 || c.Shortcuts.Folders[0] != filepath.Dir(c.Path) {
		t.Errorf("shortcuts = %+v", c.Shortcuts)
	}

	c, err = Load(write(t, "command: a.exe\nshortcuts: { mode: run }\nprofiles: { DEV: {}, VR: {} }\n"), "VR")
	if err != nil {
		t.Fatal(err)
	}
	if c.Shortcuts.Name != "game - VR" {
		t.Errorf("name = %q", c.Shortcuts.Name)
	}
}

func TestShortcutsExpand(t *testing.T) {
	c, err := Load(write(t, "name: dcs\ncommand: a.exe\nshortcuts: { mode: run, name: 'DCS ${profile}', folder: links }\nprofiles: { DEV: {} }\n"), "DEV")
	if err != nil {
		t.Fatal(err)
	}
	if c.Shortcuts.Name != "DCS DEV" || len(c.Shortcuts.Folders) != 1 || c.Shortcuts.Folders[0] != filepath.Join(filepath.Dir(c.Path), "links") {
		t.Errorf("shortcuts = %+v", c.Shortcuts)
	}
}

func TestShortcutsInSeveralFolders(t *testing.T) {
	c, err := Load(write(t, "command: a.exe\nshortcuts: { mode: run, folder: [links, '${start_menu}'] }\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	if f := c.Shortcuts.Folders; len(f) != 2 || f[0] != filepath.Join(filepath.Dir(c.Path), "links") || !strings.HasSuffix(f[1], `\Programs`) {
		t.Errorf("folders = %v", f)
	}
}

func TestShortcutsErrors(t *testing.T) {
	cases := map[string]string{
		"command: a.exe\nshortcuts: { name: x }\n":                "shortcuts.mode",
		"command: a.exe\nshortcuts: { mode: start }\n":            "shortcuts.mode",
		"command: a.exe\nshortcuts: { mode: run, name: 'a:b' }\n": "shortcuts.name",
		"command: a.exe\nvars: { desktop: x }\n":                  "réservé",
	}
	for yaml, want := range cases {
		if _, err := Load(write(t, yaml), ""); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q : %v, voulu %q", yaml, err, want)
		}
	}
}

func TestKnownFolderVariables(t *testing.T) {
	c, err := Load(write(t, "command: a.exe\nvars: { d: '${desktop}', s: '${start_menu}' }\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"d", "s"} {
		if v := c.Vars[k]; !filepath.IsAbs(v) {
			t.Errorf("%s = %q", k, v)
		}
	}
	if !strings.HasSuffix(c.Vars["s"], `\Programs`) {
		t.Errorf("start_menu = %q", c.Vars["s"])
	}
}
