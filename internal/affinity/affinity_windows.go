package affinity

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	getProcessAffinityMask = kernel32.NewProc("GetProcessAffinityMask")
	setProcessAffinityMask = kernel32.NewProc("SetProcessAffinityMask")
)

// System returns the cores of this machine that pm may use.
func System() (Mask, error) {
	_, sys, err := masks(windows.CurrentProcess())
	return sys, err
}

// Get returns the affinity of a process opened with
// PROCESS_QUERY_LIMITED_INFORMATION.
func Get(h windows.Handle) (Mask, error) {
	m, _, err := masks(h)
	return m, err
}

// Set changes the affinity of a process opened with PROCESS_SET_INFORMATION.
// Windows resets the affinity of every thread of the process to the new mask.
func Set(h windows.Handle, m Mask) error {
	if r, _, err := setProcessAffinityMask.Call(uintptr(h), uintptr(m)); r == 0 {
		return err
	}
	return nil
}

func masks(h windows.Handle) (process, system Mask, err error) {
	var p, s uintptr
	if r, _, e := getProcessAffinityMask.Call(uintptr(h), uintptr(unsafe.Pointer(&p)), uintptr(unsafe.Pointer(&s))); r == 0 {
		return 0, 0, e
	}
	return Mask(p), Mask(s), nil
}
