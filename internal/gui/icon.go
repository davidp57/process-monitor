package gui

import (
	"image/png"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/davidp57/process-monitor/internal/tray"
)

// icon serves a profile's icon (.ico or .png) as a PNG thumbnail, for the
// page to show it next to the profile.
func icon(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	// A local file only: opening \\host\share\x.ico would send the user's
	// Windows credentials to that host.
	local := filepath.IsAbs(path) && len(filepath.VolumeName(path)) == 2 // a drive letter
	if ext := strings.ToLower(filepath.Ext(path)); !local || ext != ".ico" && ext != ".png" {
		http.NotFound(w, r)
		return
	}
	img, err := tray.LoadImage(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img)
}
