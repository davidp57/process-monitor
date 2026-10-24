package gui

import (
	"net/http"

	"github.com/davidp57/process-monitor/internal/filedialog"
	"github.com/davidp57/process-monitor/internal/i18n"
)

// pickOptions is the Windows window a path field opens, by what the field
// holds: a program, an icon, a config to add or to create, a folder, or
// any file (save: one that may not exist yet).
func pickOptions(kind, start string) (filedialog.Options, error) {
	all := filedialog.Filter{Name: i18n.T("gui.pick_all"), Pattern: "*.*"}
	o := filedialog.Options{Start: start}
	switch kind {
	case "program":
		o.Filters = []filedialog.Filter{{Name: i18n.T("gui.pick_programs"), Pattern: "*.exe;*.bat;*.cmd"}, all}
	case "icon":
		o.Filters = []filedialog.Filter{{Name: i18n.T("gui.pick_icons"), Pattern: "*.ico;*.png"}}
	case "yaml", "newyaml":
		o.Filters = []filedialog.Filter{{Name: i18n.T("gui.pick_configs"), Pattern: "*.yaml;*.yml"}}
		o.Save, o.Ext = kind == "newyaml", "yaml"
	case "folder":
		o.Folder = true
	case "file":
	case "save":
		o.Save = true
	default:
		return o, i18n.Errorf("gui.pick_bad_kind", kind)
	}
	return o, nil
}

// pick opens the window and answers the path chosen, or "" when it was
// cancelled.
func (s *Server) pick(w http.ResponseWriter, r *http.Request) {
	var req struct{ Kind, Start string }
	if err := read(r, &req); err != nil {
		fail(w, err)
		return
	}
	o, err := pickOptions(req.Kind, req.Start)
	if err != nil {
		fail(w, err)
		return
	}
	path, ok, err := s.opt.Pick(o)
	if err != nil {
		fail(w, err)
		return
	}
	if !ok {
		path = ""
	}
	writeJSON(w, map[string]string{"path": path})
}
