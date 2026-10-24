package gui

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/davidp57/process-monitor/internal/config"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/registry"
	"github.com/davidp57/process-monitor/internal/shortcut"
	"github.com/davidp57/process-monitor/internal/wizard"
	"github.com/davidp57/process-monitor/internal/yamledit"
)

// configView is a config as the page lists it.
type configView struct {
	Path       string        `json:"path"`
	Remembered bool          `json:"remembered"`
	Name       string        `json:"name,omitempty"`
	Profiles   []profileView `json:"profiles"`
	Mode       string        `json:"mode,omitempty"` // of its shortcuts: run or watch
	UI         string        `json:"ui,omitempty"`
	Error      string        `json:"error,omitempty"`
}

type profileView struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

// describe loads the config at path with each profile.
func describe(path string) configView {
	v := configView{Path: path, Profiles: []profileView{}}
	names, err := config.Profiles(path)
	if err != nil {
		v.Error = err.Error()
		return v
	}
	if len(names) == 0 {
		names = []string{""}
	}
	for _, p := range names {
		c, err := config.Load(path, p)
		if err != nil {
			v.Error = err.Error()
			return v
		}
		v.Name, v.UI = c.Name, c.UI
		if c.Shortcuts != nil {
			v.Mode = c.Shortcuts.Mode
		}
		if p != "" {
			v.Profiles = append(v.Profiles, profileView{Name: p, Icon: c.Icon})
		}
	}
	return v
}

func (s *Server) configRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/configs", func(w http.ResponseWriter, r *http.Request) {
		reg, err := registry.Load()
		if err != nil {
			fail(w, err)
			return
		}
		out := []configView{}
		defaults := s.opt.Defaults()
		for _, f := range reg.Known(defaults...) {
			v := describe(f.Path)
			v.Remembered = f.Remembered
			// A .yaml of a folder that pm does not load is someone else's file.
			if v.Error != "" && !f.Remembered {
				continue
			}
			out = append(out, v)
		}
		folders := reg.Folders
		if folders == nil {
			folders = []string{}
		}
		if defaults == nil {
			defaults = []string{}
		}
		writeJSON(w, map[string]any{"configs": out, "folders": folders, "defaults": defaults})
	})
	change := func(fn func(*registry.Registry, string) error) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var req struct{ Path string }
			if err := read(r, &req); err != nil {
				fail(w, err)
				return
			}
			p := strings.Trim(strings.TrimSpace(req.Path), `"`)
			var inner error
			if err := registry.Change(func(reg *registry.Registry) { inner = fn(reg, p) }); err != nil || inner != nil {
				fail(w, errors.Join(err, inner))
				return
			}
			writeJSON(w, "ok")
		}
	}
	mux.HandleFunc("POST /api/configs/add", change(func(reg *registry.Registry, p string) error {
		if _, err := os.Stat(p); err != nil {
			return i18n.Errorf("wizard.not_found", p)
		}
		reg.AddConfig(p)
		return nil
	}))
	mux.HandleFunc("POST /api/configs/forget", change(func(reg *registry.Registry, p string) error { reg.Forget(p); return nil }))
	mux.HandleFunc("POST /api/folders/add", change(func(reg *registry.Registry, p string) error {
		if info, err := os.Stat(p); err != nil || !info.IsDir() {
			return i18n.Errorf("wizard.not_found", p)
		}
		reg.AddFolder(p)
		return nil
	}))
	mux.HandleFunc("POST /api/folders/remove", change(func(reg *registry.Registry, p string) error { reg.RemoveFolder(p); return nil }))

	mux.HandleFunc("POST /api/configs/launch", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path, Profile, Mode string }
		if err := read(r, &req); err != nil {
			fail(w, err)
			return
		}
		if req.Mode != config.ModeRun && req.Mode != config.ModeWatch {
			fail(w, i18n.Errorf("config.bad_shortcut_mode", req.Mode))
			return
		}
		if _, err := config.Load(req.Path, req.Profile); err != nil {
			fail(w, err)
			return
		}
		args := []string{req.Mode, req.Path}
		if req.Profile != "" {
			args = append(args, "--profile", req.Profile)
		}
		if err := s.opt.Launch(append([]string{s.opt.Self}, args...)); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, "ok")
	})
	mux.HandleFunc("GET /api/configs/check", func(w http.ResponseWriter, r *http.Request) {
		c, err := config.Load(r.URL.Query().Get("path"), r.URL.Query().Get("profile"))
		if err != nil {
			fail(w, err)
			return
		}
		o := c.Overview()
		writeJSON(w, map[string]any{"plan": o.Text(), "overview": o})
	})
	mux.HandleFunc("GET /api/configs/logs", func(w http.ResponseWriter, r *http.Request) {
		c, err := config.Load(r.URL.Query().Get("path"), r.URL.Query().Get("profile"))
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, logFiles(c))
	})
	mux.HandleFunc("GET /api/configs/logfile", func(w http.ResponseWriter, r *http.Request) {
		c, err := config.Load(r.URL.Query().Get("path"), r.URL.Query().Get("profile"))
		if err != nil {
			fail(w, err)
			return
		}
		// Only a log of this config: the name as listed, nothing else.
		name := r.URL.Query().Get("file")
		for _, f := range logFiles(c) {
			if f.Name == name {
				data, err := os.ReadFile(filepath.Join(c.Log.Dir, name))
				if err != nil {
					fail(w, err)
					return
				}
				writeJSON(w, map[string]string{"text": string(data)})
				return
			}
		}
		fail(w, i18n.Errorf("wizard.not_found", name))
	})
	s.shortcutRoutes(mux)
	s.editRoutes(mux)
}

type logFile struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

// logFiles lists the logs of a config's session (and profile), newest first.
func logFiles(c *config.Config) []logFile {
	prefix := c.Name
	if c.Profile != "" {
		prefix += "-" + c.Profile
	}
	matches, _ := filepath.Glob(filepath.Join(c.Log.Dir, prefix+"-*.log"))
	out := []logFile{}
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil {
			out = append(out, logFile{Name: filepath.Base(m), Size: info.Size(), Modified: info.ModTime()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out
}

// launch starts a session as its own process, in a console of its own (a
// session with the notification icon gives it back at once), detached from
// the interface: quitting the interface does not stop it.
func launch(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	const createNewConsole, createNewProcessGroup = 0x10, 0x200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewConsole | createNewProcessGroup}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// shortcutPlan is what making a config's shortcuts still needs: the
// section, the ui key, an icon for the profiles without one.
type shortcutPlan struct {
	HasSection bool            `json:"hasSection"`
	HasUI      bool            `json:"hasUI"`
	Icons      []wizard.Option `json:"icons"` // the .ico files next to the config
	Profiles   []iconChoice    `json:"profiles"`
	Folders    wizard.Question `json:"folders"`
}

type iconChoice struct {
	Name    string `json:"name"`
	Current string `json:"current,omitempty"`
	Suggest string `json:"suggest,omitempty"`
}

func (s *Server) shortcutRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/configs/shortcuts", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		v := describe(path)
		if v.Error != "" {
			fail(w, errors.New(v.Error))
			return
		}
		dir := filepath.Dir(path)
		found := wizard.FindIcons(dir)
		plan := shortcutPlan{HasSection: shortcut.HasSection(path), HasUI: shortcut.HasUI(path),
			Folders: wizard.ShortcutFolders(), Profiles: []iconChoice{}}
		q := wizard.IconQuestion("", v.Name, dir, found)
		plan.Icons = q.Options[:len(found)]
		names := v.Profiles
		if len(names) == 0 {
			c, _ := config.Load(path, "")
			names = []profileView{{Name: "", Icon: c.Icon}}
		}
		for _, p := range names {
			suggest := wizard.IconQuestion(p.Name, v.Name, dir, found).Default[0]
			if suggest == wizard.IconNone {
				suggest = ""
			}
			plan.Profiles = append(plan.Profiles, iconChoice{Name: p.Name, Current: p.Icon, Suggest: suggest})
		}
		writeJSON(w, plan)
	})
	mux.HandleFunc("POST /api/configs/shortcuts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path    string
			Mode    string            // when the section is missing
			Folders []string          // idem
			UI      string            // when the ui key is missing
			Icons   map[string]string // profile → icon, for profiles without one
		}
		if err := read(r, &req); err != nil {
			fail(w, err)
			return
		}
		if !shortcut.HasSection(req.Path) {
			if err := shortcut.AddSection(req.Path, req.Mode, req.Folders); err != nil {
				fail(w, err)
				return
			}
		}
		if req.UI != "" && !shortcut.HasUI(req.Path) {
			if err := shortcut.SetUI(req.Path, req.UI); err != nil {
				fail(w, err)
				return
			}
		}
		for p, icon := range req.Icons {
			if icon == "" {
				continue
			}
			if err := shortcut.SetIcon(req.Path, p, icon); err != nil {
				fail(w, err)
				return
			}
		}
		links, err := shortcut.ForConfig(req.Path, "", s.opt.Self)
		if err != nil {
			fail(w, err)
			return
		}
		registry.Remember(req.Path)
		written, startMenu := shortcut.WriteAll(links)
		writeJSON(w, map[string]any{"written": written, "startMenu": startMenu})
	})
}

func (s *Server) editRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/configs/edit", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(r.URL.Query().Get("path"))
		if err != nil {
			fail(w, err)
			return
		}
		doc, err := yamledit.Read(data)
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, map[string]any{"doc": doc})
	})
	mux.HandleFunc("POST /api/configs/edit", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string
			Doc  map[string]any
		}
		if err := read(r, &req); err != nil {
			fail(w, err)
			return
		}
		if err := Save(req.Path, req.Doc); err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, "ok")
	})
}

// Save writes doc into the config at path, only if pm loads the result
// with every profile; the previous file is kept as .bak.
func Save(path string, doc map[string]any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := yamledit.Apply(data, doc)
	if err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".edit")
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	defer os.Remove(tmp)
	names, err := config.Profiles(tmp)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		names = []string{""}
	}
	for _, p := range names {
		if _, err := config.Load(tmp, p); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
