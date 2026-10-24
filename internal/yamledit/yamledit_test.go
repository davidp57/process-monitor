package yamledit

import (
	"strings"
	"testing"
)

const config = `# DCS par profil
name: dcs
vars:
  saved: ${env:USERPROFILE}\Saved Games\DCS   # where DCS keeps its files
command: 'C:\jeux\DCS World\bin\DCS_updater.exe'
track: dcs.exe
hooks:
  sync:
    - ${prof}\options.lua <-> ${saved}\Config\options.lua
profiles:
  # the one to develop missions
  DEV: {}
  VR: {}
`

func TestUnchangedValuesStayAsTheyAre(t *testing.T) {
	doc, err := Read([]byte(config))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Apply([]byte(config), doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# DCS par profil", "# where DCS keeps its files", "# the one to develop missions",
		`command: 'C:\jeux\DCS World\bin\DCS_updater.exe'`, `${prof}\options.lua <-> ${saved}\Config\options.lua`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("perdu %q :\n%s", want, out)
		}
	}
}

func TestEdit(t *testing.T) {
	doc, _ := Read([]byte(config))
	doc["ui"] = "tray"                                                               // added
	delete(doc, "track")                                                             // removed
	doc["command"] = `D:\DCS\bin\DCS_updater.exe`                                    // changed, quoted as before
	doc["profiles"].(map[string]any)["DEV"] = map[string]any{"icon": `C:\i\dev.ico`} // a profile filled
	out, err := Apply([]byte(config), doc)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"ui: tray", `command: 'D:\DCS\bin\DCS_updater.exe'`, "# the one to develop missions",
		"DEV:\n    icon: " + `C:\i\dev.ico`, "# where DCS keeps its files"} {
		if !strings.Contains(s, want) {
			t.Errorf("sans %q :\n%s", want, s)
		}
	}
	if strings.Contains(s, "track") {
		t.Errorf("track resté :\n%s", s)
	}
	// The order of the keys kept does not move: name before command.
	if strings.Index(s, "name:") > strings.Index(s, "command:") {
		t.Errorf("ordre :\n%s", s)
	}
}
