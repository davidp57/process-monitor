// Package registry remembers the configs pm has worked with, and the
// folders where the interface looks for more, so the interface can list
// them. It lives in %APPDATA%\pm\configs.json.
package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Registry is the content of configs.json.
type Registry struct {
	Configs []string `json:"configs"`
	Folders []string `json:"folders"`
	// Lang is pm's language when nothing nearer says (--lang, PM_LANG, the
	// config's lang): fr or en, "" for Windows'.
	Lang string `json:"lang,omitempty"`
}

// Lang is the language setting, "" when none (or unreadable).
func Lang() string {
	r, err := Load()
	if err != nil {
		return ""
	}
	return r.Lang
}

// file is where the registry lives; tests point it elsewhere.
var file = func() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pm", "configs.json"), nil
}

// Load reads the registry; a missing file is an empty registry.
func Load() (Registry, error) {
	var r Registry
	path, err := file()
	if err != nil {
		return r, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(data, &r)
}

// Save writes the registry, through a temporary file so a reader never
// sees half of it.
func (r Registry) Save() error {
	path, err := file()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	// A temporary file of this process's own: two pm starting together
	// must not write into the same one.
	tmp := path + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func has(list []string, p string) bool {
	for _, x := range list {
		if strings.EqualFold(x, p) {
			return true
		}
	}
	return false
}

func without(list []string, p string) []string {
	var out []string
	for _, x := range list {
		if !strings.EqualFold(x, p) {
			out = append(out, x)
		}
	}
	return out
}

// Remember adds a config pm used. It never fails a session: a registry
// that cannot be written only means the interface lists one config less.
func Remember(path string) {
	_ = Change(func(r *Registry) { r.Configs = add(r.Configs, path) })
}

// changing keeps this process's changes one after the other: the
// interface may remember a config while the page adds a folder, and both
// load, change and save the same file.
var changing sync.Mutex

// Change loads the registry, applies fn and saves it.
func Change(fn func(*Registry)) error {
	changing.Lock()
	defer changing.Unlock()
	r, err := Load()
	if err != nil {
		return err
	}
	fn(&r)
	return r.Save()
}

func add(list []string, p string) []string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if has(list, p) {
		return list
	}
	return append(list, p)
}

// AddConfig, Forget, AddFolder and RemoveFolder change the lists; Forget
// never touches the config file itself.
func (r *Registry) AddConfig(p string)    { r.Configs = add(r.Configs, p) }
func (r *Registry) Forget(p string)       { r.Configs = without(r.Configs, p) }
func (r *Registry) AddFolder(p string)    { r.Folders = add(r.Folders, p) }
func (r *Registry) RemoveFolder(p string) { r.Folders = without(r.Folders, p) }

// Found is a config the interface lists, and where it comes from.
type Found struct {
	Path       string
	Remembered bool // in the list; else found in a folder only
}

// Known lists the remembered configs, then the .yaml files of the folders
// given and of the default ones (one level down each), each path once,
// sorted.
func (r Registry) Known(defaults ...string) []Found {
	seen := map[string]bool{}
	var out []Found
	for _, p := range r.Configs {
		if !seen[strings.ToLower(p)] {
			seen[strings.ToLower(p)] = true
			out = append(out, Found{Path: p, Remembered: true})
		}
	}
	for _, dir := range append(append([]string(nil), r.Folders...), defaults...) {
		for _, pattern := range []string{"*.yaml", "*.yml"} {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			for _, p := range matches {
				if !seen[strings.ToLower(p)] {
					seen[strings.ToLower(p)] = true
					out = append(out, Found{Path: p})
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path) })
	return out
}

// UseFile points the registry at path until restore is called: for the
// tests of the packages that use it, never to touch the user's own.
func UseFile(path string) (restore func()) {
	old := file
	file = func() (string, error) { return path, nil }
	return func() { file = old }
}
