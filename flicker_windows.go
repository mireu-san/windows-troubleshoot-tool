//go:build windows

package main

import (
	"golang.org/x/sys/windows"
	"path/filepath"
	"time"
	"unsafe"
)

type displayRect struct{ Left, Top, Right, Bottom int32 }
type displayMonitorInfo struct {
	Size          uint32
	Monitor, Work displayRect
	Flags         uint32
}

func captureDisplaySample() (DisplaySample, error) {
	s := DisplaySample{Time: time.Now().Format(time.RFC3339Nano)}
	monitors, err := graphicsMonitors()
	if err != nil {
		return s, err
	}
	s.Monitors = monitors
	dll := windows.NewLazySystemDLL("user32.dll")
	hwnd, _, _ := dll.NewProc("GetForegroundWindow").Call()
	if hwnd == 0 {
		s.Issue = "Foreground window unavailable"
		return s, nil
	}
	dll.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&s.PID)))
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, s.PID)
	if err != nil {
		s.Issue = "Foreground process inaccessible: " + err.Error()
	} else {
		defer windows.CloseHandle(handle)
		buffer := make([]uint16, 32768)
		size := uint32(len(buffer))
		if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil {
			s.Issue = err.Error()
		} else {
			s.Process = filepath.Base(windows.UTF16ToString(buffer[:size]))
		}
	}
	var rect displayRect
	ok, _, _ := dll.NewProc("GetWindowRect").Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	monitor, _, _ := dll.NewProc("MonitorFromWindow").Call(hwnd, 2)
	info := displayMonitorInfo{}
	info.Size = uint32(unsafe.Sizeof(info))
	infoOK, _, _ := dll.NewProc("GetMonitorInfoW").Call(monitor, uintptr(unsafe.Pointer(&info)))
	if ok != 0 && infoOK != 0 {
		full := rect.Left <= info.Monitor.Left && rect.Top <= info.Monitor.Top && rect.Right >= info.Monitor.Right && rect.Bottom >= info.Monitor.Bottom
		s.Fullscreen = &full
	}
	return s, nil
}
