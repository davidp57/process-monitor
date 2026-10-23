package i18n

import "golang.org/x/sys/windows"

var getUserDefaultUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// SystemLanguage is the Windows display language, as "fr" or "en"; "" for
// any other.
func SystemLanguage() string {
	id, _, _ := getUserDefaultUILanguage.Call()
	switch id & 0x3ff { // primary language identifier
	case 0x0c:
		return "fr"
	case 0x09:
		return "en"
	}
	return ""
}
