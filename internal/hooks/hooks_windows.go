package hooks

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// noWindow keeps an exec action from opening a console window: its output
// is captured anyway, and without a console of its own (pm --tray) Windows
// would open one for every console program.
func noWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
}
