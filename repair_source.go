package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Inspect the exit status, never localized stdout (which may contain unrelated errors).
func isRepairSourceMissing(step CommandSpec, err error) bool {
	if !strings.EqualFold(step.Name, "DISM.exe") {
		return false
	}
	var status interface{ ExitCode() int }
	if !errors.As(err, &status) {
		return false
	}
	switch uint32(status.ExitCode()) {
	case 0x800f0915, 0x800f081f:
		return true
	}
	return false
}

func withRepairSource(step CommandSpec, source string) CommandSpec {
	step.Args = append(append([]string{}, step.Args...), "/Source:"+source, "/English")
	// Keep Windows Update available for content absent from the installation media.
	step.StartText = "설치 원본과 Windows Update를 사용해 구성 요소 복구를 다시 시도합니다."
	return step
}

func (a *App) resolveRepairSource(ctx context.Context, selected string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if a.findRepairSource != nil {
		return a.findRepairSource(ctx, selected)
	}
	return findSystemRepairSource(ctx, selected)
}

// The dialog is part of the operation lock, so no other servicing work can start.
func (a *App) StartRepairFromMedia() (bool, error) {
	if !a.beginOperation() {
		return false, errors.New("검사·복구 작업 또는 종료 예약이 진행 중입니다")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   translateNative("Windows 설치 원본 선택", a.GetLanguageSettings().Language),
		Filters: []runtime.FileFilter{{DisplayName: "Windows image (install.wim, install.esd)", Pattern: "*.wim;*.esd"}},
	})
	if err != nil || path == "" {
		a.endOperation()
		return false, err
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock()
	a.repairCancel = cancel
	a.shutdownAfterRepair = false
	a.mu.Unlock()
	go a.runRepairWithSource(ctx, path)
	return true, nil
}

func (a *App) OpenWindowsRepairSettings() error {
	if !a.beginOperation() {
		return errors.New("검사·복구 작업 또는 종료 예약이 진행 중입니다")
	}
	defer a.endOperation()
	return openWindowsRepairSettings()
}

type repairImage struct {
	Path         string
	Index        int
	Edition      string
	Architecture int
	Major        int
	Minor        int
	Build        int
	Revision     int
	Languages    []string
}
type repairSourceInventory struct {
	Current repairImage
	Images  []repairImage
	Issues  []string
}

func selectRepairSource(inv repairSourceInventory) (string, error) {
	current := inv.Current
	if current.Build == 0 || current.Edition == "" || len(current.Languages) == 0 {
		return "", errors.New("현재 Windows 버전 정보를 확인할 수 없습니다.")
	}
	var best *repairImage
	for i := range inv.Images {
		candidate := &inv.Images[i]
		if len(repairSourceRejections(current, *candidate)) > 0 {
			continue
		}
		if best == nil || candidate.Revision > best.Revision {
			best = candidate
		}
	}
	if best == nil {
		return "", nil
	}
	kind := "wim"
	if strings.HasSuffix(strings.ToLower(best.Path), ".esd") {
		kind = "esd"
	}
	return fmt.Sprintf("%s:%s:%d", kind, best.Path, best.Index), nil
}

// These are conservative candidate filters, not proof that every repair payload exists.
func repairSourceRejections(current, candidate repairImage) []string {
	var reasons []string
	if candidate.Index < 1 {
		reasons = append(reasons, "invalid image index")
	}
	if !strings.EqualFold(candidate.Edition, current.Edition) {
		reasons = append(reasons, "edition mismatch")
	}
	if candidate.Architecture != current.Architecture {
		reasons = append(reasons, "architecture mismatch")
	}
	if candidate.Major != current.Major || candidate.Minor != current.Minor || candidate.Build != current.Build {
		reasons = append(reasons, "build mismatch (cross-build compatibility unverified)")
	}
	if candidate.Revision < current.Revision {
		reasons = append(reasons, "source update revision is older")
	}
	for _, language := range current.Languages {
		found := false
		for _, available := range candidate.Languages {
			if strings.EqualFold(language, available) {
				found = true
				break
			}
		}
		if !found {
			reasons = append(reasons, "missing language: "+language)
		}
	}
	lower := strings.ToLower(candidate.Path)
	if !strings.HasSuffix(lower, ".wim") && !strings.HasSuffix(lower, ".esd") {
		reasons = append(reasons, "unsupported image format")
	}
	return reasons
}

func repairSourceFailure(inv repairSourceInventory) error {
	var details strings.Builder
	fmt.Fprintf(&details, "Target: %d.%d.%d.%d; edition=%s; architecture=%d; languages=%s\n", inv.Current.Major, inv.Current.Minor, inv.Current.Build, inv.Current.Revision, inv.Current.Edition, inv.Current.Architecture, strings.Join(inv.Current.Languages, ", "))
	if len(inv.Images) == 0 {
		details.WriteString("No installation image could be inspected.\n")
	}
	for _, candidate := range inv.Images {
		fmt.Fprintf(&details, "%s [index %d, %d.%d.%d.%d]: %s\n", candidate.Path, candidate.Index, candidate.Major, candidate.Minor, candidate.Build, candidate.Revision, strings.Join(repairSourceRejections(inv.Current, candidate), "; "))
	}
	for _, issue := range inv.Issues {
		fmt.Fprintf(&details, "Inspection failed: %s\n", issue)
	}
	return errors.New(strings.TrimSpace(details.String()))
}
