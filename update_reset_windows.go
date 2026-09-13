//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type updateReporter func(stage, title, message, output string, progress int)

func resetWindowsUpdate(ctx context.Context, report updateReporter) (resultErr error) {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("Windows 서비스 관리자 연결 실패: %w", err)
	}
	defer manager.Disconnect()

	services := []string{"bits", "wuauserv", "cryptsvc"}
	servicesToRestart := make([]string, 0, len(services))
	// Restore every service that this operation actually stopped, even when a
	// later step fails. Services that were already stopped remain untouched.
	defer func() {
		for index := len(servicesToRestart) - 1; index >= 0; index-- {
			name := servicesToRestart[index]
			if err := setServiceState(context.Background(), manager, name, true); err != nil && resultErr == nil {
				resultErr = fmt.Errorf("%s 서비스 재시작 실패: %w", name, err)
			}
		}
	}()

	report("stopping", "업데이트 서비스 중지 중", "캐시를 안전하게 초기화하기 위해 관련 서비스를 잠시 중지합니다.", "서비스 중지를 시작합니다.", 12)
	for index, name := range services {
		state, err := queryServiceState(manager, name)
		if err != nil {
			return fmt.Errorf("%s 서비스 상태 확인 실패: %w", name, err)
		}
		if state != svc.Stopped {
			if err := setServiceState(ctx, manager, name, false); err != nil {
				return fmt.Errorf("%s 서비스 중지 실패: %w", name, err)
			}
			if state == svc.Running || state == svc.StartPending {
				servicesToRestart = append(servicesToRestart, name)
			}
		}
		report("stopping", "업데이트 서비스 중지 중", "캐시를 안전하게 초기화하기 위해 관련 서비스를 잠시 중지합니다.", name+" 서비스를 중지했습니다.", 18+index*7)
	}

	systemRoot := filepath.Clean(os.Getenv("SystemRoot"))
	if systemRoot == "." || !filepath.IsAbs(systemRoot) {
		return errors.New("Windows 시스템 폴더를 확인할 수 없습니다")
	}

	report("resetting", "업데이트 캐시 백업 중", "기존 캐시를 삭제하지 않고 백업 이름으로 변경합니다.", "캐시 폴더를 확인합니다.", 45)
	timestamp := time.Now().Format("20060102-150405")
	directories := []string{
		filepath.Join(systemRoot, "SoftwareDistribution"),
		filepath.Join(systemRoot, "System32", "catroot2"),
	}
	for index, source := range directories {
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			report("resetting", "업데이트 캐시 백업 중", "기존 캐시를 삭제하지 않고 백업 이름으로 변경합니다.", filepath.Base(source)+" 폴더가 없어 건너뜁니다.", 55+index*12)
			continue
		} else if err != nil {
			return fmt.Errorf("%s 확인 실패: %w", source, err)
		}
		backup := source + ".bak-" + timestamp
		if err := os.Rename(source, backup); err != nil {
			return fmt.Errorf("%s 백업 실패: %w", filepath.Base(source), err)
		}
		report("resetting", "업데이트 캐시 백업 중", "기존 캐시를 삭제하지 않고 백업 이름으로 변경합니다.", filepath.Base(source)+" 폴더를 "+filepath.Base(backup)+"(으)로 백업했습니다.", 55+index*12)
	}

	report("starting", "업데이트 서비스 다시 시작 중", "Windows 업데이트 기능을 다시 시작하고 있습니다.", "서비스를 다시 시작합니다.", 82)
	return nil
}

func queryServiceState(manager *mgr.Mgr, name string) (svc.State, error) {
	service, err := manager.OpenService(name)
	if err != nil {
		return 0, err
	}
	defer service.Close()
	status, err := service.Query()
	return status.State, err
}

func setServiceState(ctx context.Context, manager *mgr.Mgr, name string, shouldRun bool) error {
	service, err := manager.OpenService(name)
	if err != nil {
		return err
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return err
	}
	if shouldRun && status.State == svc.Running {
		return nil
	}
	if !shouldRun && status.State == svc.Stopped {
		return nil
	}

	if shouldRun {
		if status.State != svc.StartPending {
			if err := service.Start(); err != nil {
				return err
			}
		}
	} else {
		if status.State != svc.StopPending {
			if _, err := service.Control(svc.Stop); err != nil {
				return err
			}
		}
	}

	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	target := svc.Stopped
	if shouldRun {
		target = svc.Running
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return errors.New("서비스 상태 변경 시간 초과")
		case <-ticker.C:
			status, err = service.Query()
			if err != nil {
				return err
			}
			if status.State == target {
				return nil
			}
		}
	}
}
