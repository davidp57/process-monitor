package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMenuArgs(t *testing.T) {
	if got := menuArgs("run", `C:\p\dcs.yaml`, "VR"); !reflect.DeepEqual(got, []string{"run", `C:\p\dcs.yaml`, "--profile", "VR"}) {
		t.Errorf("%v", got)
	}
	if got := menuArgs("shortcut", `C:\p\dcs.yaml`, ""); !reflect.DeepEqual(got, []string{"shortcut", `C:\p\dcs.yaml`}) {
		t.Errorf("%v", got)
	}
	// What the menu builds, the command line parses back the same way.
	a, err := parse(menuArgs("watch", `C:\p\sim.yaml`, "DEV"))
	if err != nil || a.cmd != "watch" || a.path != `C:\p\sim.yaml` || a.profile != "DEV" {
		t.Errorf("%+v, %v", a, err)
	}
}

// pm.exe usually sits in a pm folder next to its configs: the menu suggests
// one found there or one level up.
func TestNearbyConfig(t *testing.T) {
	d := t.TempDir()
	exeDir := filepath.Join(d, "pm")
	os.MkdirAll(exeDir, 0o755)
	if got := nearbyConfig(exeDir); got != "" {
		t.Errorf("rien à trouver : %q", got)
	}
	os.WriteFile(filepath.Join(d, "pm-dcs.yaml"), nil, 0o644)
	if got := nearbyConfig(exeDir); got != filepath.Join(d, "pm-dcs.yaml") {
		t.Errorf("au-dessus : %q", got)
	}
	os.WriteFile(filepath.Join(exeDir, "local.yaml"), nil, 0o644)
	if got := nearbyConfig(exeDir); got != filepath.Join(exeDir, "local.yaml") {
		t.Errorf("à côté : %q", got)
	}
}
