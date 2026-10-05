//go:build windows

package main

import (
	"testing"
	"unsafe"
)

func TestGraphicsWindowsStructLayouts(t *testing.T) {
	if unsafe.Sizeof(graphicsDisplayDevice{}) != 840 {
		t.Fatal("DISPLAY_DEVICEW layout")
	}
	if unsafe.Sizeof(graphicsDevMode{}) != 220 {
		t.Fatal("DEVMODEW layout")
	}
	var mode graphicsDevMode
	if unsafe.Offsetof(mode.Width) != 172 || unsafe.Offsetof(mode.Frequency) != 184 {
		t.Fatal("DEVMODEW offsets")
	}
}
