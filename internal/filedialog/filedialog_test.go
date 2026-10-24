package filedialog

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

func TestStart(t *testing.T) {
	d := t.TempDir()
	file := filepath.Join(d, "dcs.yaml")
	os.WriteFile(file, nil, 0o644)
	for _, c := range []struct {
		o            Options
		folder, name string
	}{
		{Options{}, "", ""},
		{Options{Start: d}, d, ""},
		{Options{Start: file}, d, "dcs.yaml"},
		{Options{Start: filepath.Join(d, "new.yaml"), Save: true}, d, "new.yaml"},
		{Options{Start: filepath.Join(d, "missing", "x.exe")}, "", "x.exe"},
		{Options{Start: file, Folder: true}, d, ""},
		{Options{Start: filepath.Join(d, "missing", "x"), Folder: true}, "", ""},
	} {
		if f, n := start(c.o); f != c.folder || n != c.name {
			t.Errorf("%+v : %q %q, attendu %q %q", c.o, f, n, c.folder, c.name)
		}
	}
}

// The window is made and set up, never shown: what Windows reads back is
// what was asked, so each method was called at its right place.
func TestPrepare(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		t.Fatal(err)
	}
	defer ole.CoUninitialize()
	dir := t.TempDir()
	for _, c := range []struct {
		o    Options
		want uint32
		name string
	}{
		{Options{Save: true, Start: filepath.Join(dir, "new.yaml"), Title: "t", Ext: "yaml", Filters: []Filter{{"Configs", "*.yaml"}}}, fosOverwritePrompt, "new.yaml"},
		{Options{Start: filepath.Join(dir, "x.exe"), Filters: []Filter{{"Programs", "*.exe"}, {"All", "*.*"}}}, fosFileMustExist, "x.exe"},
		{Options{Folder: true, Start: dir}, fosPickFolders, ""},
	} {
		d, err := prepare(c.o)
		if err != nil {
			t.Fatal(err)
		}
		var opts uint32
		call(d.p, mGetOptions, uintptr(unsafe.Pointer(&opts)))
		if opts&c.want == 0 || opts&fosForceFileSystem == 0 {
			t.Errorf("%+v : options %#x", c.o, opts)
		}
		var name *uint16
		if hr := call(d.p, mGetFileName, uintptr(unsafe.Pointer(&name))); hr == 0 && name != nil {
			if got := windows.UTF16PtrToString(name); got != c.name {
				t.Errorf("%+v : nom %q, attendu %q", c.o, got, c.name)
			}
			windows.CoTaskMemFree(unsafe.Pointer(name))
		} else if c.name != "" {
			t.Errorf("%+v : pas de nom (%#x)", c.o, hr)
		}
		var folder unsafe.Pointer
		if hr := call(d.p, mGetFolder, uintptr(unsafe.Pointer(&folder))); hr != 0 {
			t.Errorf("%+v : pas de dossier (%#x)", c.o, hr)
		} else {
			var p *uint16
			call(folder, mDisplayName, sigdnFileSysPath, uintptr(unsafe.Pointer(&p)))
			if got := windows.UTF16PtrToString(p); !strings.EqualFold(got, dir) {
				t.Errorf("%+v : dossier %q, attendu %q", c.o, got, dir)
			}
			windows.CoTaskMemFree(unsafe.Pointer(p))
			call(folder, mRelease)
		}
		d.release()
	}
}
