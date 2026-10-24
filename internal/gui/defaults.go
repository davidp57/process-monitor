package gui

import (
	"os"
	"path/filepath"
)

// defaultFolders are looked into without being given: the folder of pm.exe
// and the one above (where its configs usually are), and where the wizard
// puts a config by default: Saved Games\DCS\Profiles (and DCS.openbeta),
// and each folder of Documents\pm. Only the ones that exist.
func defaultFolders(self string) []string {
	var out []string
	add := func(p string) {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			out = append(out, p)
		}
	}
	if self != "" {
		add(filepath.Dir(self))
		add(filepath.Dir(filepath.Dir(self)))
	}
	home := os.Getenv("USERPROFILE")
	for _, d := range []string{"DCS", "DCS.openbeta"} {
		add(filepath.Join(home, "Saved Games", d, "Profiles"))
	}
	if docs, err := documents(); err == nil {
		entries, _ := os.ReadDir(filepath.Join(docs, "pm"))
		for _, e := range entries {
			if e.IsDir() {
				add(filepath.Join(docs, "pm", e.Name()))
			}
		}
	}
	return out
}
