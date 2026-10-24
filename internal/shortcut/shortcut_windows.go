package shortcut

import (
	"runtime"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

// Window styles of a shortcut, as WScript.Shell names them.
const (
	styleNormal    = 1
	styleMinimized = 7
)

// withShortcut opens the shortcut at path through the Shell's COM object
// (created when missing) and hands it to fn.
func withShortcut(path string, fn func(sc *ole.IDispatch) error) error {
	runtime.LockOSThread() // COM is initialised per thread
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		// S_FALSE: already initialised on this thread, which is fine.
		if oe, ok := err.(*ole.OleError); !ok || oe.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()
	unk, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unk.Release()
	shell, err := unk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer shell.Release()
	v, err := oleutil.CallMethod(shell, "CreateShortcut", path)
	if err != nil {
		return err
	}
	sc := v.ToIDispatch()
	defer sc.Release()
	return fn(sc)
}

func save(l Link) error {
	return withShortcut(l.Path, func(sc *ole.IDispatch) error {
		style := styleNormal
		if l.Minimized {
			style = styleMinimized
		}
		props := []struct {
			name  string
			value any
		}{
			{"TargetPath", l.Target},
			{"Arguments", l.Args},
			{"WorkingDirectory", l.Workdir},
			{"WindowStyle", style},
		}
		if l.Icon != "" {
			props = append(props, struct {
				name  string
				value any
			}{"IconLocation", l.Icon + ",0"})
		}
		for _, p := range props {
			if _, err := oleutil.PutProperty(sc, p.name, p.value); err != nil {
				return err
			}
		}
		_, err := oleutil.CallMethod(sc, "Save")
		return err
	})
}

// Read loads the shortcut at path.
func Read(path string) (Link, error) {
	l := Link{Path: path}
	err := withShortcut(path, func(sc *ole.IDispatch) error {
		get := func(name string) (*ole.VARIANT, error) { return oleutil.GetProperty(sc, name) }
		for name, dst := range map[string]*string{"TargetPath": &l.Target, "Arguments": &l.Args,
			"WorkingDirectory": &l.Workdir, "IconLocation": &l.Icon} {
			v, err := get(name)
			if err != nil {
				return err
			}
			*dst = v.ToString()
		}
		l.Icon, _, _ = strings.Cut(l.Icon, ",") // "path,index"
		v, err := get("WindowStyle")
		if err != nil {
			return err
		}
		l.Minimized = v.Val == styleMinimized
		return nil
	})
	return l, err
}

// startMenuDir is the user's Start menu Programs folder, or "".
func startMenuDir() string {
	p, _ := windows.KnownFolderPath(windows.FOLDERID_Programs, 0)
	return p
}
