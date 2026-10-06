//go:build windows

package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
)

//go:embed repair/find-source.ps1
var repairSourceScript string

func findSystemRepairSource(ctx context.Context, selected string) (string, error) {
	units := utf16.Encode([]rune(repairSourceScript))
	encoded := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(encoded[i*2:], unit)
	}
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, path, "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(encoded))
	// The selected path is data, never executable PowerShell or cmd.exe text.
	cmd.Env = append(os.Environ(), "REPAIR_SELECTED_IMAGE="+selected)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", fmt.Errorf("복구 원본 조회 실패: %w", err)
	}
	var inv repairSourceInventory
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(out), "\ufeff")), &inv); err != nil {
		return "", err
	}
	source, err := selectRepairSource(inv)
	if source == "" && err == nil {
		return "", repairSourceFailure(inv)
	}
	return source, err
}

func openWindowsRepairSettings() error { return openWindowsURI("ms-settings:recovery") }
