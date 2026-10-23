package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// interactive reports whether stdin is a console a person can type in.
func interactive() bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(os.Stdin.Fd()), &mode) == nil
}
