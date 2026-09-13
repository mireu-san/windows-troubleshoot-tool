//go:build windows

package main

import "golang.org/x/sys/windows"

var userDefaultUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

func systemLanguage() string {
	id, _, _ := userDefaultUILanguage.Call()
	switch uint16(id) & 0x3ff {
	case 0x12:
		return "ko"
	case 0x11:
		return "ja"
	default:
		return "en"
	}
}
