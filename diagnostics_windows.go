//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func expandDiagnosticArchive(path string) ([]string, func(), error) {
	dir, err := os.MkdirTemp("", "repair-cbs-")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(os.Getenv("SystemRoot"), "System32", "expand.exe"), "-F:*.log", path, dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err = cmd.Run(); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("expand: %w", err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(paths) == 0 {
		cleanup()
		return nil, func() {}, fmt.Errorf("archive has no readable .log files")
	}
	return paths, cleanup, nil
}
