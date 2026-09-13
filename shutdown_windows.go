//go:build windows

package main

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

func runShutdownCommand(abort bool) error {
	args := []string{"/s", "/t", "300", "/c", nativeText("Windows 검사 및 복구 완료: 5분 뒤 종료합니다. 작업 중인 파일을 저장해 주세요.")}
	if abort {
		args = []string{"/a"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "shutdown.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		if abort {
			if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1116 {
				return nil // No shutdown is pending (for example, cancelled in Windows).
			}
			return fmt.Errorf("종료 예약을 취소하지 못했습니다. Windows에서 shutdown /a로 취소해 주세요: %w", err)
		}
		return fmt.Errorf("Windows 종료 예약 명령이 실패했습니다: %w", err)
	}
	return nil
}
