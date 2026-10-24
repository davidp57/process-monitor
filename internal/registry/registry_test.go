package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func use(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	old := file
	file = func() (string, error) { return filepath.Join(d, "pm", "configs.json"), nil }
	t.Cleanup(func() { file = old })
	return d
}

func TestRememberAndKnown(t *testing.T) {
	d := use(t)
	if r, err := Load(); err != nil || len(r.Known()) != 0 {
		t.Fatalf("vide : %+v, %v", r, err)
	}
	a := filepath.Join(d, "b", "pm-dcs.yaml")
	Remember(a)
	Remember(filepath.Join(d, "B", "PM-DCS.yaml")) // the same file, Windows does not mind the case
	folder := filepath.Join(d, "folder")
	os.MkdirAll(filepath.Join(folder, "deep"), 0o755)
	for _, f := range []string{"sim.yaml", "other.yml", "notes.txt", `deep\hidden.yaml`} {
		os.WriteFile(filepath.Join(folder, f), nil, 0o644)
	}
	if err := Change(func(r *Registry) { r.AddFolder(folder); r.AddConfig(filepath.Join(folder, "sim.yaml")) }); err != nil {
		t.Fatal(err)
	}
	r, _ := Load()
	got := r.Known()
	want := []Found{{a, true}, {filepath.Join(folder, "other.yml"), false}, {filepath.Join(folder, "sim.yaml"), true}}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d : %+v, voulu %+v", i, got[i], want[i])
		}
	}

	// Forgetting a config never touches the file; removing a folder hides its files.
	Change(func(r *Registry) { r.Forget(a); r.RemoveFolder(folder) })
	r, _ = Load()
	if k := r.Known(); len(k) != 1 || k[0].Path != filepath.Join(folder, "sim.yaml") {
		t.Errorf("%+v", k)
	}
}

// Default folders are looked into without being kept in the file.
func TestKnownWithDefaults(t *testing.T) {
	d := use(t)
	def := filepath.Join(d, "Profiles")
	os.MkdirAll(def, 0o755)
	os.WriteFile(filepath.Join(def, "pm-dcs.yaml"), nil, 0o644)
	r, _ := Load()
	if k := r.Known(def); len(k) != 1 || k[0].Path != filepath.Join(def, "pm-dcs.yaml") || k[0].Remembered {
		t.Errorf("%+v", k)
	}
	if r2, _ := Load(); len(r2.Folders) != 0 {
		t.Errorf("dossier par défaut écrit : %+v", r2)
	}
}

func TestLangSetting(t *testing.T) {
	use(t)
	if Lang() != "" {
		t.Error("langue sans réglage")
	}
	Change(func(r *Registry) { r.Lang = "fr" })
	Remember(`C:\x.yaml`) // other changes keep it
	if Lang() != "fr" {
		t.Errorf("%q", Lang())
	}
}

func TestChangesTogether(t *testing.T) {
	t.Cleanup(UseFile(filepath.Join(t.TempDir(), "configs.json")))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Remember(fmt.Sprintf(`C:\cfg\%d.yaml`, i))
		}()
	}
	wg.Wait()
	r, err := Load()
	if err != nil || len(r.Configs) != 20 {
		t.Errorf("%d configs gardées sur 20 : %v", len(r.Configs), err)
	}
}
