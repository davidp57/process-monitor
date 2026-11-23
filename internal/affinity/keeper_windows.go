package affinity

import (
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/davidp57/process-monitor/internal/i18n"
)

// Keeper holds processes found by image name on a set of cores, for as
// long as Sweep is called: a process that starts, or starts again, gets
// the mask on the next sweep. Restore gives each one back the affinity it
// had when it was first seen.
//
// Sweep and Restore are called from one goroutine.
type Keeper struct {
	Mask  Mask
	Names []string
	// Program is the launched program's mask. A process first seen with it
	// inherited it (the program started it): Restore gives it every core
	// instead, not the program's.
	Program Mask
	// Event reports what was done, one line per process.
	Event func(msg string)

	seen map[uint32]*kept
}

type kept struct {
	name     string
	created  windows.Filetime // tells a process from a later one with the same PID
	original Mask
	failed   bool // told once, not tried again
	told     bool // its placing was logged
}

const keeperAccess = windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.PROCESS_SET_INFORMATION

// Sweep gives the mask to every named process that does not have it.
func (k *Keeper) Sweep() {
	if k.seen == nil {
		k.seen = map[uint32]*kept{}
	}
	procs, err := list(k.Names)
	if err != nil {
		return // the next sweep tries again
	}
	alive := map[uint32]bool{}
	for pid, name := range procs {
		alive[pid] = true
		k.sweepOne(pid, name)
	}
	for pid := range k.seen {
		if !alive[pid] {
			delete(k.seen, pid)
		}
	}
}

func (k *Keeper) sweepOne(pid uint32, name string) {
	h, err := windows.OpenProcess(keeperAccess, false, pid)
	if err != nil {
		if e := k.seen[pid]; e == nil {
			k.seen[pid] = &kept{name: name, failed: true}
			k.event(i18n.T("affinity.others_failed", name, pid, err))
		}
		return
	}
	defer windows.CloseHandle(h)
	created := creationTime(h)
	e := k.seen[pid]
	if e != nil && e.created != created {
		e = nil // the PID now names another process
	}
	if e != nil && e.failed {
		return
	}
	cur, err := Get(h)
	if err != nil {
		return
	}
	if e == nil {
		original := cur
		if k.Program != 0 && cur == k.Program {
			if sys, err := System(); err == nil {
				original = sys
			}
		}
		e = &kept{name: name, created: created, original: original}
		k.seen[pid] = e
	}
	if cur == k.Mask {
		return
	}
	if err := Set(h, k.Mask); err != nil {
		e.failed = true
		k.event(i18n.T("affinity.others_failed", name, pid, err))
		return
	}
	if !e.told {
		e.told = true
		k.event(i18n.T("affinity.others_set", name, pid, k.Mask))
	}
}

// Held reports whether a process has been placed on the mask and is still
// running, as of the last sweep.
func (k *Keeper) Held() bool {
	for _, e := range k.seen {
		if e.told && !e.failed {
			return true
		}
	}
	return false
}

// Restore gives the processes still running their original affinity back,
// unless something else changed it since.
func (k *Keeper) Restore() {
	for pid, e := range k.seen {
		if e.failed || e.original == k.Mask {
			continue
		}
		h, err := windows.OpenProcess(keeperAccess, false, pid)
		if err != nil {
			continue // gone
		}
		if creationTime(h) == e.created {
			if cur, err := Get(h); err == nil && cur == k.Mask {
				if err := Set(h, e.original); err != nil {
					k.event(i18n.T("affinity.restore_failed", e.name, pid, err))
				} else {
					k.event(i18n.T("affinity.restored", e.name, pid, e.original))
				}
			}
		}
		windows.CloseHandle(h)
	}
	k.seen = nil
}

func (k *Keeper) event(msg string) {
	if k.Event != nil {
		k.Event(msg)
	}
}

// list returns the running processes whose image name is one of names.
func list(names []string) (map[uint32]string, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	found := map[uint32]string{}
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		exe := windows.UTF16ToString(e.ExeFile[:])
		for _, n := range names {
			if strings.EqualFold(exe, n) {
				found[e.ProcessID] = exe
				break
			}
		}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return found, nil
}

func creationTime(h windows.Handle) windows.Filetime {
	var c, x, k, u windows.Filetime
	_ = windows.GetProcessTimes(h, &c, &x, &k, &u)
	return c
}
