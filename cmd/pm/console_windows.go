package main

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// interactive reports whether stdin is a console a person can type in.
func interactive() bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &mode) == nil
}

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	getConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	freeConsole           = kernel32.NewProc("FreeConsole")
)

// releaseConsole detaches pm from its console when no other process shares
// it (pm was started from a shortcut): Windows then closes the window. In a
// terminal the console is shared with the shell, and pm keeps it. It reports
// whether the console was released.
func releaseConsole() bool {
	if !consoleAlone() {
		return false
	}
	r, _, _ := freeConsole.Call()
	return r != 0
}

// consoleAlone reports whether pm is the only process of its console: it
// was started from Explorer (a double-click, a shortcut), not from a shell.
func consoleAlone() bool {
	var ids [2]uint32
	n, _, _ := getConsoleProcessList.Call(uintptr(unsafe.Pointer(&ids[0])), uintptr(len(ids)))
	return n == 1
}
