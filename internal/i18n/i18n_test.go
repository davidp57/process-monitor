package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestCatalogsHaveTheSameKeys(t *testing.T) {
	for key := range catalogs["fr"] {
		if _, ok := catalogs["en"][key]; !ok {
			t.Errorf("%s : absente de en.json", key)
		}
	}
	for key := range catalogs["en"] {
		if _, ok := catalogs["fr"][key]; !ok {
			t.Errorf("%s : absente de fr.json", key)
		}
	}
	if len(catalogs["fr"]) < 50 {
		t.Errorf("catalogue suspect : %d clés", len(catalogs["fr"]))
	}
}

var verb = regexp.MustCompile(`%[-+# 0]*[0-9]*(?:\.[0-9]+)?[a-zA-Z%]`)

// A translation with other format verbs than the original would print
// garbage, or panic-free nonsense like %!d(string=…).
func TestFormatVerbsMatch(t *testing.T) {
	for key, fr := range catalogs["fr"] {
		en := catalogs["en"][key]
		if a, b := verb.FindAllString(fr, -1), verb.FindAllString(en, -1); strings.Join(a, " ") != strings.Join(b, " ") {
			t.Errorf("%s : fr %v, en %v", key, a, b)
		}
	}
}

// goFiles lists the non-test Go files of the module, outside this package's catalogs.
func goFiles(t *testing.T) []string {
	var files []string
	for _, root := range []string{"../../cmd", "../../internal"} {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
				files = append(files, p)
			}
			return nil
		})
	}
	if len(files) < 10 {
		t.Fatalf("sources introuvables : %v", files)
	}
	return files
}

var call = regexp.MustCompile(`i18n\.(?:T|Errorf)\("([^"]+)"`)

func TestEveryKeyUsedExists(t *testing.T) {
	used := map[string]bool{}
	for _, f := range goFiles(t) {
		src, _ := os.ReadFile(f)
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			if strings.HasSuffix(m[1], ".") {
				continue // a prefix completed at run time ("status." + id): see dynamic
			}
			used[m[1]] = true
			if _, ok := catalogs["en"][m[1]]; !ok {
				t.Errorf("%s : clé %q absente du catalogue", f, m[1])
			}
		}
	}
	var unused []string
	for key := range catalogs["en"] {
		if !used[key] && !dynamic(key) {
			unused = append(unused, key)
		}
	}
	sort.Strings(unused)
	if len(unused) > 0 {
		t.Errorf("clés jamais utilisées : %v", unused)
	}
}

// dynamic keys are built at run time (a prefix plus a value), so the scan
// above cannot see them.
func dynamic(key string) bool {
	for _, prefix := range []string{"status.", "ready.", "test."} {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

var accented = regexp.MustCompile(`"[^"\n]*[àâäéèêëîïôöùûüçÀÂÉÈÊÎÔÙÛÇ][^"\n]*"`)

// Every message goes through the catalog: an accented literal left in the
// code is a French message the English user would get.
func TestNoFrenchLiteralInCode(t *testing.T) {
	for _, f := range goFiles(t) {
		src, _ := os.ReadFile(f)
		for i, line := range strings.Split(string(src), "\n") {
			code, _, _ := strings.Cut(line, "//")
			if accented.MatchString(code) {
				t.Errorf("%s:%d : message en dur : %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// sink matches a message written straight into an output call, whatever its
// language: accents or not, a word there means a message the catalog missed.
var sink = regexp.MustCompile(`(?:Infof|Warnf|Errorf|Info|Warn|Error|errors\.New|Sprintf|Printf|Println|Print|Fprintln|Fprintf|Fprint)\(\s*(?:os\.Std\w+,\s*)?"([^"]*)"`)

var word = regexp.MustCompile(`[A-Za-z]{3,}`)

// technical formats are not messages: a file name, and a build mistake
// shown before any language exists.
var technical = map[string]bool{
	"%s-%s.log":            true,
	"locales/%s.json : %v": true,
}

func TestNoMessageLiteralInOutputCalls(t *testing.T) {
	for _, f := range goFiles(t) {
		src, _ := os.ReadFile(f)
		for i, line := range strings.Split(string(src), "\n") {
			code, _, _ := strings.Cut(line, "//")
			for _, m := range sink.FindAllStringSubmatchIndex(code, -1) {
				if strings.HasSuffix(code[:m[0]], "i18n.") {
					continue // i18n.Errorf("key"): the catalog itself
				}
				if msg := code[m[2]:m[3]]; word.MatchString(msg) && !technical[msg] {
					t.Errorf("%s:%d : message en dur %q", f, i+1, msg)
				}
			}
		}
	}
}

func TestTranslateAndFallback(t *testing.T) {
	defer Set("en")
	Set("fr")
	if got := T("test.hello", "David"); got != "Bonjour David" {
		t.Errorf("fr : %q", got)
	}
	Set("en")
	if got := T("test.hello", "David"); got != "Hello David" {
		t.Errorf("en : %q", got)
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Errorf("clé inconnue : %q", got)
	}
}

func TestResolveOrder(t *testing.T) {
	sys := func() string { return "fr-FR" }
	cases := []struct{ flag, env, cfg, want string }{
		{"en", "fr", "fr", "en"},
		{"", "EN", "fr", "en"},
		{"", "", "en-US", "en"},
		{"", "", "", "fr"},
		{"de", "", "", "fr"}, // unsupported: next source
	}
	for _, c := range cases {
		if got := Resolve(c.flag, c.env, c.cfg, sys); got != c.want {
			t.Errorf("%+v : %q", c, got)
		}
	}
	if got := Resolve("", "", "", func() string { return "de-DE" }); got != "en" {
		t.Errorf("repli : %q", got)
	}
}
