package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var varRef = regexp.MustCompile(`\$\{([^}]+)\}`)

// expander resolves ${name} from vars (which may reference each other),
// the built-ins profile, name and config_dir, and ${env:NAME}.
type expander struct {
	vars     map[string]string
	resolved map[string]string
	visiting map[string]bool
}

func (e *expander) lookup(key string) (string, error) {
	if env, ok := strings.CutPrefix(key, "env:"); ok {
		v, found := os.LookupEnv(env)
		if !found {
			return "", fmt.Errorf("variable d'environnement %s absente", env)
		}
		return v, nil
	}
	if v, ok := e.resolved[key]; ok {
		return v, nil
	}
	raw, ok := e.vars[key]
	if !ok {
		return "", fmt.Errorf("variable ${%s} non définie", key)
	}
	if e.visiting[key] {
		return "", fmt.Errorf("variable ${%s} définie en boucle", key)
	}
	e.visiting[key] = true
	v, err := e.expand(raw)
	delete(e.visiting, key)
	if err != nil {
		return "", err
	}
	e.resolved[key] = v
	return v, nil
}

func (e *expander) expand(s string) (string, error) {
	var firstErr error
	out := varRef.ReplaceAllStringFunc(s, func(m string) string {
		v, err := e.lookup(m[2 : len(m)-1])
		if err != nil && firstErr == nil {
			firstErr = err
		}
		return v
	})
	return out, firstErr
}

func (c *Config) expand() error {
	vars := map[string]string{}
	for k, v := range c.Vars {
		switch k {
		case "profile", "name", "config_dir":
			return fmt.Errorf("vars : %q est réservé (fourni par pm)", k)
		}
		vars[k] = v
	}
	if len(c.Profiles) > 0 {
		vars["profile"] = c.Profile // without profiles, ${profile} is an error, never ""
	}
	vars["name"] = c.Name
	vars["config_dir"] = filepath.Dir(c.Path)
	e := &expander{vars: vars, resolved: map[string]string{}, visiting: map[string]bool{}}

	var firstErr error
	x := func(s *string) {
		v, err := e.expand(*s)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		*s = v
	}
	xs := func(list []string) {
		for i := range list {
			x(&list[i])
		}
	}
	// xp expands a path and cleans it, so "${prof}\file" never yields "dir\\file".
	// A relative path is taken from the config's folder, not from wherever
	// pm was started.
	xp := func(s *string) {
		x(s)
		if *s != "" {
			*s = c.abs(*s)
		}
	}
	xa := func(actions []Action) {
		for i := range actions {
			if actions[i].Copy != nil {
				xp(&actions[i].Copy.From)
				xp(&actions[i].Copy.To)
			}
			xp(&actions[i].Delete)
			xs(actions[i].Exec)
		}
	}

	x(&c.Command) // a bare name is looked up in PATH: leave it alone
	xs(c.Args)
	xp(&c.Workdir)
	x(&c.Track)
	for k, v := range c.Env {
		x(&v)
		c.Env[k] = v
	}
	if c.Ready != nil {
		xp(&c.Ready.File)
	}
	xa(c.Hooks.Before)
	xa(c.Hooks.After)
	xs(c.Hooks.Sync)
	xp(&c.Log.Dir)
	c.Vars = map[string]string{}
	for k := range vars {
		v, err := e.lookup(k)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		c.Vars[k] = v
	}
	return firstErr
}

// abs cleans path and anchors it to the config's folder when relative.
func (c *Config) abs(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(c.Path), path)
	}
	return filepath.Clean(path)
}
