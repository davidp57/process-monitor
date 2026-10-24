package config

import "golang.org/x/sys/windows"

var knownFolderIDs = map[string]*windows.KNOWNFOLDERID{
	"desktop":    windows.FOLDERID_Desktop,
	"start_menu": windows.FOLDERID_Programs,
}

// knownFolder resolves a built-in variable naming a Windows known folder.
// One Windows cannot find is not found: using it is then the usual
// "undefined variable" error, not a silent empty path.
func knownFolder(name string) (string, bool) {
	id, ok := knownFolderIDs[name]
	if !ok {
		return "", false
	}
	p, err := windows.KnownFolderPath(id, 0)
	return p, err == nil
}
