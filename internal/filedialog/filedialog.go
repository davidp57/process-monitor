// Package filedialog opens the Windows Open, Save as and folder windows,
// for the interface: a browser page never learns where a file it is given
// lives, pm does.
package filedialog

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

// Options says which window to open.
type Options struct {
	Folder  bool     // pick a folder rather than a file
	Save    bool     // Save as: the file may not exist yet
	Title   string   // the window's title
	Start   string   // a path already typed: the window opens there
	Filters []Filter // file types offered, the first chosen
	Ext     string   // Save as: the extension added to a name typed without one
}

// Filter is one file type: "Programs" and "*.exe".
type Filter struct{ Name, Pattern string }

// ErrBusy: one window at a time.
var ErrBusy error = busy{}

type busy struct{}

func (busy) Error() string { return i18n.T("gui.pick_busy") }

var mu sync.Mutex

var (
	clsidOpen = ole.NewGUID("{DC1C5A9C-E88A-4dde-A5A1-60F82A20AEF7}")
	clsidSave = ole.NewGUID("{C0B4E2F3-BA21-4773-8DBA-335EC946EB8B}")
	iidOpen   = ole.NewGUID("{d57c7288-d4ad-4768-be02-9d969532d960}")
	iidSave   = ole.NewGUID("{84bccd23-5fde-4cdb-aea4-af64b83d78ab}")
	iidItem   = ole.NewGUID("{43826d1e-e718-42ee-bc55-a1e261c37bfe}")

	shell32      = windows.NewLazySystemDLL("shell32.dll")
	user32       = windows.NewLazySystemDLL("user32.dll")
	itemFromPath = shell32.NewProc("SHCreateItemFromParsingName")
	createWindow = user32.NewProc("CreateWindowExW")
	destroyWin   = user32.NewProc("DestroyWindow")
	foreground   = user32.NewProc("SetForegroundWindow")
)

// IFileDialog methods, by their place in its table (after IUnknown's 3
// and IModalWindow's Show).
const (
	mShow        = 3
	mSetTypes    = 4
	mSetOptions  = 9
	mGetOptions  = 10
	mSetFolder   = 12
	mGetFolder   = 13
	mSetFileName = 15
	mGetFileName = 16
	mSetTitle    = 17
	mGetResult   = 20
	mSetExt      = 22
	mRelease     = 2
	mDisplayName = 5 // IShellItem.GetDisplayName
)

const (
	fosOverwritePrompt = 0x2
	fosNoChangeDir     = 0x8
	fosPickFolders     = 0x20
	fosForceFileSystem = 0x40
	fosPathMustExist   = 0x800
	fosFileMustExist   = 0x1000

	sigdnFileSysPath = 0x80058000
	errCancelled     = 0x800704C7 // HRESULT_FROM_WIN32(ERROR_CANCELLED)
)

// Show opens the window and waits for it: the path chosen, or ok false
// when it was cancelled.
func Show(o Options) (path string, ok bool, err error) {
	if !mu.TryLock() {
		return "", false, ErrBusy
	}
	defer mu.Unlock()
	runtime.LockOSThread() // COM is initialised per thread
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if oe, isOle := err.(*ole.OleError); !isOle || oe.Code() != 1 { // S_FALSE: already done
			return "", false, err
		}
	}
	defer ole.CoUninitialize()
	d, err := prepare(o)
	if err != nil {
		return "", false, err
	}
	defer d.release()

	// pm is not the program in front, so a window of its own would open
	// behind the browser: a hidden topmost owner brings it to the front.
	owner, _, _ := createWindow.Call(0x80|0x8, d.str("STATIC"), 0, 0x80000000, 0, 0, 0, 0, 0, 0, 0, 0) // TOOLWINDOW|TOPMOST, POPUP
	if owner != 0 {
		foreground.Call(owner)
		defer destroyWin.Call(owner)
	}
	if hr := call(d.p, mShow, owner); hr != 0 {
		if uint32(hr) == errCancelled {
			return "", false, nil
		}
		return "", false, ole.NewError(hr)
	}
	var item unsafe.Pointer
	if hr := call(d.p, mGetResult, uintptr(unsafe.Pointer(&item))); hr != 0 {
		return "", false, ole.NewError(hr)
	}
	defer call(item, mRelease)
	var p *uint16
	if hr := call(item, mDisplayName, sigdnFileSysPath, uintptr(unsafe.Pointer(&p))); hr != 0 {
		return "", false, ole.NewError(hr)
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(p))
	return windows.UTF16PtrToString(p), true, nil
}

// dialog is a window made and set up, not shown yet.
type dialog struct {
	p    unsafe.Pointer // the IFileDialog
	keep [][]uint16     // the strings handed to Windows, kept alive with it
}

func (d *dialog) str(s string) uintptr {
	u, _ := windows.UTF16FromString(s)
	d.keep = append(d.keep, u)
	return uintptr(unsafe.Pointer(&u[0]))
}

func (d *dialog) release() {
	call(d.p, mRelease)
	runtime.KeepAlive(d.keep)
}

// prepare makes the window and sets it up as o says.
func prepare(o Options) (*dialog, error) {
	clsid, iid := clsidOpen, iidOpen
	if o.Save {
		clsid, iid = clsidSave, iidSave
	}
	unk, err := ole.CreateInstance(clsid, iid)
	if err != nil {
		return nil, err
	}
	d := &dialog{p: unsafe.Pointer(unk)}
	var opts uint32
	call(d.p, mGetOptions, uintptr(unsafe.Pointer(&opts)))
	opts |= fosForceFileSystem | fosNoChangeDir | fosPathMustExist
	switch {
	case o.Folder:
		opts |= fosPickFolders
	case o.Save:
		opts |= fosOverwritePrompt
	default:
		opts |= fosFileMustExist
	}
	call(d.p, mSetOptions, uintptr(opts))
	if o.Title != "" {
		call(d.p, mSetTitle, d.str(o.Title))
	}
	if len(o.Filters) > 0 && !o.Folder {
		type spec struct{ name, pattern uintptr }
		specs := make([]spec, len(o.Filters))
		for i, f := range o.Filters {
			specs[i] = spec{d.str(f.Name), d.str(f.Pattern)}
		}
		call(d.p, mSetTypes, uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0])))
		runtime.KeepAlive(specs)
	}
	if o.Ext != "" {
		call(d.p, mSetExt, d.str(o.Ext))
	}
	folder, name := start(o)
	if folder != "" {
		var item unsafe.Pointer
		if hr, _, _ := itemFromPath.Call(d.str(folder), 0, uintptr(unsafe.Pointer(iidItem)), uintptr(unsafe.Pointer(&item))); hr == 0 {
			call(d.p, mSetFolder, uintptr(item))
			call(item, mRelease)
		}
	}
	if name != "" {
		call(d.p, mSetFileName, d.str(name))
	}
	return d, nil
}

// start is where the window opens: the folder typed, or the folder of
// the file typed with its name filled in; nothing when neither exists.
func start(o Options) (folder, name string) {
	if o.Start == "" {
		return "", ""
	}
	p := filepath.Clean(o.Start)
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p, ""
	}
	if o.Folder {
		p = filepath.Dir(p)
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p, ""
		}
		return "", ""
	}
	if st, err := os.Stat(filepath.Dir(p)); err == nil && st.IsDir() {
		return filepath.Dir(p), filepath.Base(p)
	}
	return "", filepath.Base(p)
}

// call calls the method at index m of the COM object obj.
func call(obj unsafe.Pointer, m int, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, m*int(unsafe.Sizeof(uintptr(0)))))
	hr, _, _ := syscall.SyscallN(fn, append([]uintptr{uintptr(obj)}, args...)...)
	return hr
}
