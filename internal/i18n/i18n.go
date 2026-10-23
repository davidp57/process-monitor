// Package i18n holds pm's messages in French and English and picks the
// language. Every message shown to the user goes through T.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
)

//go:embed locales/*.json
var files embed.FS

// catalogs maps a language to its messages, by key.
var catalogs = map[string]map[string]string{}

var current atomic.Value

func init() {
	for _, lang := range []string{"fr", "en"} {
		data, err := files.ReadFile("locales/" + lang + ".json")
		if err != nil {
			panic(err) // embedded: a missing catalog is a build mistake
		}
		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			panic(fmt.Sprintf("locales/%s.json : %v", lang, err))
		}
		catalogs[lang] = m
	}
	current.Store("en")
}

// Set selects the language; an unsupported one is ignored.
func Set(lang string) {
	if l := normalize(lang); l != "" {
		current.Store(l)
	}
}

// Lang is the language in use.
func Lang() string { return current.Load().(string) }

// T returns the message for key in the current language, formatted with
// args. A key missing in that language falls back to English, then to the
// key itself: a missing entry never hides a message.
func T(key string, args ...any) string {
	msg, ok := catalogs[Lang()][key]
	if !ok {
		if msg, ok = catalogs["en"][key]; !ok {
			msg = key
		}
	}
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

// Errorf is fmt.Errorf on a catalog message: %w in it wraps an error as usual.
func Errorf(key string, args ...any) error {
	msg, ok := catalogs[Lang()][key]
	if !ok {
		if msg, ok = catalogs["en"][key]; !ok {
			msg = key
		}
	}
	return fmt.Errorf(msg, args...)
}

// Resolve picks the language: the --lang flag, else PM_LANG, else the
// config's lang, else the system's UI language, else English. A value that
// is not a supported language is skipped.
func Resolve(flag, env, config string, system func() string) string {
	for _, v := range []string{flag, env, config, system()} {
		if l := normalize(v); l != "" {
			return l
		}
	}
	return "en"
}

// normalize turns "fr", "FR", "fr-FR" or "fr_CA" into "fr"; "" when unsupported.
func normalize(v string) string {
	l := strings.ToLower(strings.TrimSpace(v))
	if i := strings.IndexAny(l, "-_"); i > 0 {
		l = l[:i]
	}
	if _, ok := catalogs[l]; ok {
		return l
	}
	return ""
}
