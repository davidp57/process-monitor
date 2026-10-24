package wizard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Snapshot is the content of the watched folders: every file's path and the
// hash of its content. Comparing hashes, not dates: a game may rewrite a
// file on every exit without changing it.
type Snapshot map[string]string

// Take reads every file under roots, except the ignored ones (full paths).
// A root that does not exist is empty.
func Take(roots, ignore []string) (Snapshot, error) {
	skip := map[string]bool{}
	for _, p := range ignore {
		skip[strings.ToLower(filepath.Clean(p))] = true
	}
	s := Snapshot{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if errors.Is(err, fs.ErrNotExist) && p == root {
				return filepath.SkipDir
			}
			if err != nil || d.IsDir() || skip[strings.ToLower(p)] {
				return err
			}
			h, err := hash(p)
			if err != nil {
				return err
			}
			s[p] = h
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

func hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Changes lists what differs from before to after: path → true for a file
// changed or created, false for a file removed.
func Changes(before, after Snapshot) map[string]bool {
	out := map[string]bool{}
	for p, h := range after {
		if before[p] != h {
			out[p] = true
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out[p] = false
		}
	}
	return out
}

// Paths returns the snapshot's paths, sorted.
func (s Snapshot) Paths() []string {
	out := make([]string, 0, len(s))
	for p := range s {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// storePath is where the copy of a live file is kept under a store folder:
// the drive letter becomes a folder, C:\a\b → <store>\C\a\b.
func storePath(store, live string) string {
	return filepath.Join(store, strings.Replace(filepath.Clean(live), ":", "", 1))
}

// Store copies the given live files into store.
func Store(store string, paths []string) error {
	for _, p := range paths {
		if err := copyFile(p, storePath(store, p)); err != nil {
			return err
		}
	}
	return nil
}

// manifest is what the original state needs to be put back, even by a
// later pm new after this one was interrupted.
type manifest struct {
	Roots    []string `json:"roots"`
	Ignore   []string `json:"ignore,omitempty"`
	Original Snapshot `json:"original"`
}

const manifestName = "original.json"

// SaveOriginal takes the original state of roots, keeps a copy of every
// file under dir\original and writes the manifest that Restore reads. The
// ignored files are never saved, compared or put back: the game owns them.
func SaveOriginal(dir string, roots, ignore []string) (Snapshot, error) {
	s, err := Take(roots, ignore)
	if err != nil {
		return nil, err
	}
	if err := Store(filepath.Join(dir, "original"), s.Paths()); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(manifest{Roots: roots, Ignore: ignore, Original: s}, "", "  ")
	if err != nil {
		return nil, err
	}
	return s, os.WriteFile(filepath.Join(dir, manifestName), data, 0o644)
}

// HasOriginal reports whether dir holds an original state to put back.
func HasOriginal(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, manifestName))
	return err == nil
}

func readManifest(dir string) (manifest, error) {
	var m manifest
	data, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil {
		return m, err
	}
	return m, json.Unmarshal(data, &m)
}

// Restore puts the watched folders back in their original state: files
// that were added are removed, files that changed or went missing are
// copied back. Files already as they were are not touched.
func Restore(dir string) error {
	m, err := readManifest(dir)
	if err != nil {
		return err
	}
	now, err := Take(m.Roots, m.Ignore)
	if err != nil {
		return err
	}
	var errs []error
	for p, present := range Changes(m.Original, now) {
		if _, was := m.Original[p]; present && !was {
			errs = append(errs, os.Remove(p))
			continue
		}
		errs = append(errs, copyFile(storePath(filepath.Join(dir, "original"), p), p))
	}
	return errors.Join(errs...)
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o644)
}

// FindIcons lists the .ico files in dir and its folders down to three
// levels, the wizard's own copies left out: where a player keeps the icons
// of the profiles next to the config.
func FindIcons(dir string) []string {
	var out []string
	base := len(strings.Split(filepath.Clean(dir), string(filepath.Separator)))
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".pm-new" || len(strings.Split(p, string(filepath.Separator)))-base >= 3 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".ico") {
			out = append(out, p)
		}
		return nil
	})
	return out
}
