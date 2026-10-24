package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"

	"github.com/davidp57/process-monitor/internal/i18n"
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
	return writeShortcuts(links)
}

func writeShortcuts(links []shortcut.Link) int {
	rc := 0
	startMenu, _ := windows.KnownFolderPath(windows.FOLDERID_Programs, 0)
	inStartMenu := false
	for _, l := range links {
		kept, err := shortcut.Write(l)
		if err != nil {
			fmt.Fprintln(os.Stderr, i18n.T("shortcut.failed", l.Path, err))
			rc = 1
			continue
		}
		fmt.Println(i18n.T("shortcut.written", l.Path, l.Args))
		if kept != "" {
			fmt.Println(i18n.T("shortcut.kept", kept))
		}
		inStartMenu = inStartMenu || (startMenu != "" && strings.EqualFold(filepath.Dir(l.Path), startMenu))
	}
	// Windows lets no program pin to the taskbar without asking; from the
	// Start menu, it is two clicks for the player.
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
