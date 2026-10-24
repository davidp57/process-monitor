package wizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, p, s string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		return "<absent>"
	}
	return string(b)
}

func TestChangesComparesContentNotDates(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "Config")
	opts, joy, gone := filepath.Join(root, "options.lua"), filepath.Join(root, "Input", "joy.lua"), filepath.Join(root, "old.lua")
	put(t, opts, "a")
	put(t, gone, "x")
	before, err := Take([]string{root, filepath.Join(d, "missing")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	put(t, opts, "a") // rewritten, same content
	put(t, joy, "new")
	os.Remove(gone)
	after, _ := Take([]string{root}, nil)
	got := Changes(before, after)
	if len(got) != 2 || got[joy] != true || got[gone] != false {
		t.Errorf("changes = %v", got)
	}
}

// An ignored file (DCS's login data) is never saved, compared or put back:
// an old copy would replace the fresh one the game just wrote.
func TestIgnoredFileIsLeftAlone(t *testing.T) {
	d := t.TempDir()
	root, wiz := filepath.Join(d, "Config"), filepath.Join(d, ".pm-new")
	auth, opts := filepath.Join(root, "authdata.bin"), filepath.Join(root, "options.lua")
	put(t, auth, "old token")
	put(t, opts, "original")
	s, err := SaveOriginal(wiz, []string{root}, []string{strings.ToUpper(auth)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s[auth]; ok || len(s) != 1 {
		t.Errorf("photo : %v", s)
	}
	put(t, auth, "fresh token")
	put(t, opts, "changed")
	if err := Restore(wiz); err != nil {
		t.Fatal(err)
	}
	if read(t, auth) != "fresh token" || read(t, opts) != "original" {
		t.Errorf("auth %q, options %q", read(t, auth), read(t, opts))
	}
}

func TestRestorePutsTheOriginalBack(t *testing.T) {
	d := t.TempDir()
	root, wiz := filepath.Join(d, "Config"), filepath.Join(d, ".pm-new")
	opts, kept, removed, added := filepath.Join(root, "options.lua"), filepath.Join(root, "kept.lua"),
		filepath.Join(root, "removed.lua"), filepath.Join(root, "Input", "added.lua")
	put(t, opts, "original")
	put(t, kept, "same")
	put(t, removed, "here")
	if _, err := SaveOriginal(wiz, []string{root}, nil); err != nil {
		t.Fatal(err)
	}
	if !HasOriginal(wiz) {
		t.Fatal("pas de manifeste")
	}
	put(t, opts, "changed by the game")
	os.Remove(removed)
	put(t, added, "new device")
	info, _ := os.Stat(kept)

	if err := Restore(wiz); err != nil {
		t.Fatal(err)
	}
	if read(t, opts) != "original" || read(t, removed) != "here" || read(t, added) != "<absent>" {
		t.Errorf("options %q, removed %q, added %q", read(t, opts), read(t, removed), read(t, added))
	}
	if after, _ := os.Stat(kept); !after.ModTime().Equal(info.ModTime()) {
		t.Error("un fichier inchangé a été réécrit")
	}
}
