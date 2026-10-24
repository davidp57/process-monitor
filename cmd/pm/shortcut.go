package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/registry"
	"github.com/davidp57/process-monitor/internal/shortcut"
)

// shortcuts is pm shortcut: (re)creates the shortcuts of a config, all
// pointing at this pm.exe.
func shortcuts(a args) int {
	links, err := shortcut.ForConfig(a.path, a.profile, self())
	if err != nil {
		fmt.Fprintln(os.Stderr, i18n.T("cli.config_error", err))
		return 2
	}
	registry.Remember(a.path)
	return writeShortcuts(links)
}

func writeShortcuts(links []shortcut.Link) int {
	rc := 0
	written, inStartMenu := shortcut.WriteAll(links)
	for _, w := range written {
		if w.Err != "" {
			fmt.Fprintln(os.Stderr, i18n.T("shortcut.failed", w.Path, w.Err))
			rc = 1
			continue
		}
		fmt.Println(i18n.T("shortcut.written", w.Path, w.Args))
		if w.Kept != "" {
			fmt.Println(i18n.T("shortcut.kept", w.Kept))
		}
	}
	if inStartMenu {
		fmt.Println(i18n.T("shortcut.taskbar_hint"))
	}
	return rc
}

// self is the absolute path of the running pm.exe: the shortcuts start it
// where it is now, so moving it means running pm shortcut again.
func self() string {
	exe, err := os.Executable()
	if err != nil {
		return "pm.exe"
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		return real
	}
	return exe
}
