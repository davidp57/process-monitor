package lock

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// open keeps the file open for writing without write sharing: a second
// writer gets a sharing violation, readers can still see who holds it.
func open(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), path), nil
}

func isBusy(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
