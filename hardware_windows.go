//go:build windows

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed graphics/hardware.ps1
var hardwareScript string

func collectHardware(ctx context.Context, start, end time.Time, request HardwareRequest) (*HardwareReport, error) {
	incident := ""
	if t, err := time.Parse(time.RFC3339, request.Incident); err == nil {
		incident = t.Format(time.RFC3339)
	}
	script := strings.NewReplacer("__START__", start.Format(time.RFC3339), "__END__", end.Format(time.RFC3339), "__INCIDENT__", incident).Replace(hardwareScript)
	data, err := graphicsPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	var r HardwareReport
	if err = json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("하드웨어 수집 결과 해석 실패: %w", err)
	}
	if r.Sections == nil {
		r.Sections = make(map[string]HardwareSection)
	}
	monitors, err := graphicsMonitors()
	if err != nil {
		r.Issues = append(r.Issues, err.Error())
	} else {
		// Omit PnP identifiers, which can contain monitor serial identifiers.
		for i := range monitors {
			monitors[i].ID = ""
			monitors[i].AdapterID = ""
		}
		raw, _ := json.Marshal(monitors)
		r.Sections["화면 모드·배치"] = HardwareSection{"수집 완료", raw}
	}
	return &r, nil
}
func hardwareDesktop(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	data, err := graphicsPowerShell(ctx, "[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); [Environment]::GetFolderPath([Environment+SpecialFolder]::DesktopDirectory)")
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(data))
	if path == "" {
		return "", fmt.Errorf("바탕화면 경로가 비어 있습니다")
	}
	return path, nil
}
func openHardwareFolder(folder string) error {
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "explorer.exe"), folder)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
