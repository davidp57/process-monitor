package main

import (
	"fmt"
	"os"

	"github.com/davidp57/process-monitor/internal/gui"
	"github.com/davidp57/process-monitor/internal/i18n"
	"github.com/davidp57/process-monitor/internal/tray"
)

// guiMain is pm gui, and a double-click on pm.exe: the interface in the
// browser, with its notification icon. Started from Explorer, the console
// is given back at once: its window closes.
func guiMain() int {
	console := !releaseConsole()
	if err := gui.Run(self()); err != nil {
		if console {
			fmt.Fprintln(os.Stderr, i18n.T("gui.failed", err))
		} else {
			tray.Alert("pm", i18n.T("gui.failed", err))
		}
		return 1
	}
	return 0
}
