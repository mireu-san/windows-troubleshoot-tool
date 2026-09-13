//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

func openWindowsURI(uri string) error {
	cmd := exec.Command("explorer.exe", uri)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Windows 기능을 열 수 없습니다: %w", err)
	}
	// Explorer may return exit code 1 after successfully dispatching a URI.
	go func() { _ = cmd.Wait() }()
	return nil
}

func openWindowsUpdateTroubleshooter() error {
	return openWindowsURI("ms-contact-support://smc-to-emerald/WindowsUpdateTroubleshooter")
}

func openWindowsUpdateSettings() error {
	return openWindowsURI("ms-settings:windowsupdate")
}
