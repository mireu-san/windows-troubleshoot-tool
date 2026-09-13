//go:build windows

package main

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
)

func runDefenderQuickScan(ctx context.Context) error {
	// Use a fixed command, without changing Defender settings or exclusions.
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-Command",
		"$ErrorActionPreference = 'Stop'; try { Start-MpScan -ScanType QuickScan -ErrorAction Stop } catch { exit 1 }")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return errors.New("빠른 검사를 완료하지 못했습니다. Windows 보안에서 Defender가 활성화되어 있는지, 다른 검사가 진행 중인지 확인해 주세요")
	}
	return nil
}

func openDefenderSecurity() error { return openWindowsURI("windowsdefender://threat") }
