package gui

import (
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestIconLocalFilesOnly(t *testing.T) {
	p := filepath.Join(t.TempDir(), "dev.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	f.Close()
	for path, want := range map[string]int{
		p: http.StatusOK,
		// The same file, through a path Windows may resolve on the network.
		`\\?\` + p:                  http.StatusNotFound,
		`\\127.0.0.1\share\dev.png`: http.StatusNotFound,
		"dev.png":                   http.StatusNotFound,
	} {
		rec := httptest.NewRecorder()
		icon(rec, httptest.NewRequest("GET", "/api/icon?"+url.Values{"path": {path}}.Encode(), nil))
		if rec.Code != want {
			t.Errorf("%s : %d, attendu %d", path, rec.Code, want)
		}
	}
}
