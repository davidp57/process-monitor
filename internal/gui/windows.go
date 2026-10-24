package gui

import "golang.org/x/sys/windows"

// documents is the user's Documents folder, where the wizard suggests
// configs that have no better place.
func documents() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
}
