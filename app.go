package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const microsoftHelpURL = "https://support.microsoft.com/ko-kr/windows/experience/backup-recovery/use-the-system-file-checker-tool-to-repair-missing-or-corrupted-system-files"

type RepairEvent struct {
	Stage             string `json:"stage"`
	Title             string `json:"title"`
	Message           string `json:"message"`
	Output            string `json:"output,omitempty"`
	Progress          int    `json:"progress"`
	ShutdownScheduled bool   `json:"shutdownScheduled,omitempty"`
}

type CommandSpec struct {
	Name        string
	Args        []string
	Stage       string
	Title       string
	StartText   string
	DoneText    string
	ProgressMin int
	ProgressMax int
}

type commandRunner interface {
	Run(context.Context, CommandSpec, func(string, int)) error
}

type App struct {
	languageMu          sync.RWMutex
	languagePreference  string
	ctx                 context.Context
	runner              commandRunner
	emitEvent           func(RepairEvent)
	mu                  sync.Mutex
	running             bool
	appDownloadURL      string
	appUpdateDigest     string
	appUpdateName       string
	stagedUpdate        string
	downloadingUpdate   bool
	installingUpdate    bool
	repairCancel        context.CancelFunc
	shutdownAfterRepair bool
	shutdownPending     bool
	shutdownCommand     func(bool) error
}

func NewApp() *App {
	return &App{runner: newSystemRunner(), languagePreference: readLanguagePreference()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emitEvent = func(event RepairEvent) {
		runtime.EventsEmit(ctx, "repair:event", event)
	}
}

// StartRepair starts the Microsoft-recommended DISM then SFC sequence.
// It returns immediately and publishes repair:event updates while work continues.
func (a *App) StartRepair() error { return a.StartRepairWithShutdown(false) }

func (a *App) StartRepairWithShutdown(shutdown bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running || a.installingUpdate || a.shutdownPending {
		return errors.New("검사·복구 작업 또는 종료 예약이 진행 중입니다")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.running = true
	a.repairCancel = cancel
	a.shutdownAfterRepair = shutdown
	go a.runRepair(ctx)
	return nil
}

func (a *App) CancelRepair() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.repairCancel != nil {
		a.repairCancel()
	}
}

// ResetWindowsUpdate safely resets the local Windows Update download cache.
func (a *App) ResetWindowsUpdate() error {
	if !a.beginOperation() {
		return errors.New("검사·복구 작업 또는 종료 예약이 진행 중입니다")
	}
	go a.runWindowsUpdateReset()
	return nil
}

func (a *App) beginOperation() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running || a.installingUpdate || a.shutdownPending {
		return false
	}
	a.running = true
	return true
}

func (a *App) endOperation() {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
}

func (a *App) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *App) beforeClose(ctx context.Context) bool {
	if !a.IsRunning() {
		return false
	}
	language := a.GetLanguageSettings().Language
	waitButton := translateNative("계속 기다리기", language)
	closeButton := translateNative("지금 종료", language)
	selection, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type:          runtime.WarningDialog,
		Title:         translateNative("검사 또는 복구 작업이 진행 중입니다", language),
		Message:       translateNative("급히 나가야 한다면 지금 종료할 수 있습니다. 진행 중인 작업은 완료되지 않았을 수 있으며, 이미 적용된 변경은 되돌리지 않습니다.\n\n시스템 검사 및 복구는 취소 버튼으로 중단을 확인한 뒤 닫는 것을 권장합니다. 업데이트 캐시 초기화 중이라면 서비스 복원이 끝날 때까지 기다려 주세요.\n\n다음에 앱을 열어 다시 검사할 수 있습니다.", language),
		Buttons:       []string{waitButton, closeButton},
		DefaultButton: waitButton,
		CancelButton:  waitButton,
	})
	if err != nil {
		return true
	}
	return selection != closeButton
}

// OpenRecoveryHelp opens documentation; it does not attempt offline repair.
func (a *App) OpenRecoveryHelp() {
	runtime.BrowserOpenURL(a.ctx, "https://support.microsoft.com/en-us/windows/experience/startup-boot/startup-repair")
}

func (a *App) OpenMicrosoftHelp() {
	runtime.BrowserOpenURL(a.ctx, microsoftHelpURL)
}

// OpenQuickAssist launches the Microsoft Quick Assist app included with Windows 11.
func (a *App) OpenQuickAssist() error {
	return openQuickAssist()
}

// OpenWindowsUpdateTroubleshooter opens Microsoft's automated troubleshooter
// in the Windows Get Help app.
func (a *App) OpenWindowsUpdateTroubleshooter() error {
	return openWindowsUpdateTroubleshooter()
}

// OpenWindowsUpdate opens the Windows Update settings page.
func (a *App) OpenWindowsUpdate() error {
	return openWindowsUpdateSettings()
}

func (a *App) StartDefenderQuickScan() error {
	if !a.beginOperation() {
		return errors.New("검사·복구 작업 또는 종료 예약이 진행 중입니다. 작업을 마치고 종료 예약을 취소한 뒤 다시 시도해 주세요")
	}
	defer a.endOperation()
	return runDefenderQuickScan(a.ctx)
}

func (a *App) OpenDefenderSecurity() error { return openDefenderSecurity() }

func (a *App) runRepair(ctx context.Context) {
	defer func() {
		a.mu.Lock()
		if a.repairCancel != nil {
			a.repairCancel()
		}
		a.repairCancel = nil
		a.shutdownAfterRepair = false
		a.running = false
		a.mu.Unlock()
	}()
	cancelled := func() bool {
		if ctx.Err() == nil {
			return false
		}
		a.emit(RepairEvent{Stage: "cancelled", Title: "검사와 복구를 취소했습니다", Message: "다음 복구 단계는 실행하지 않습니다. 이미 적용된 변경은 유지됩니다. 필요하면 Windows를 다시 시작한 뒤 다시 검사해 주세요."})
		return true
	}

	steps := []CommandSpec{
		{
			Name: "DISM.exe", Args: []string{"/Online", "/Cleanup-Image", "/RestoreHealth"},
			Stage: "dism", Title: "Windows 구성 요소 복구 중",
			StartText: "Windows Update에서 정상 파일을 확인하고 있습니다. 인터넷 연결이 필요할 수 있습니다.",
			DoneText:  "Windows 구성 요소 복구를 마쳤습니다.", ProgressMin: 4, ProgressMax: 52,
		},
		{
			Name: "sfc.exe", Args: []string{"/scannow"},
			Stage: "sfc", Title: "시스템 파일 검사 중",
			StartText: "보호된 시스템 파일을 검사하고 손상된 파일을 복구하고 있습니다.",
			DoneText:  "시스템 파일 검사를 마쳤습니다.", ProgressMin: 54, ProgressMax: 98,
		},
	}

	started := time.Now()
	for _, step := range steps {
		if cancelled() {
			return
		}
		a.emit(RepairEvent{Stage: step.Stage, Title: step.Title, Message: step.StartText, Progress: step.ProgressMin})
		err := a.runner.Run(ctx, step, func(line string, progress int) {
			a.emit(RepairEvent{Stage: step.Stage, Title: step.Title, Message: step.StartText, Output: line, Progress: progress})
		})
		if cancelled() {
			return
		}
		if err != nil {
			a.emit(RepairEvent{
				Stage: "error", Title: "복구를 완료하지 못했습니다",
				Message: friendlyError(step, err), Output: err.Error(), Progress: step.ProgressMin,
			})
			return
		}
		a.emit(RepairEvent{Stage: step.Stage, Title: step.Title, Message: step.DoneText, Progress: step.ProgressMax})
	}

	if cancelled() {
		return
	}
	duration := time.Since(started).Round(time.Second)
	message := fmt.Sprintf("전체 작업 시간은 약 %s입니다. 문제가 계속되면 Windows를 다시 시작해 주세요.", koreanDuration(duration))
	scheduled, err := a.scheduleRepairShutdown(ctx)
	if err != nil {
		message += " 자동 종료를 예약하지 못했습니다: " + err.Error()
	} else if scheduled {
		message += " 5분 뒤 컴퓨터가 종료됩니다. 작업 중인 파일을 저장해 주세요. 앱을 닫아도 예약은 유지됩니다."
	}
	a.emit(RepairEvent{
		Stage: "complete", Title: "검사와 복구가 완료되었습니다",
		Message:           message,
		ShutdownScheduled: scheduled,
		Progress:          100,
	})
}

func (a *App) runWindowsUpdateReset() {
	defer a.endOperation()
	a.emitUpdate(RepairEvent{
		Stage: "preparing", Title: "업데이트 초기화 준비 중",
		Message: "Windows 업데이트 관련 서비스를 안전하게 확인하고 있습니다.", Progress: 4,
	})
	err := resetWindowsUpdate(a.ctx, func(stage, title, message, output string, progress int) {
		a.emitUpdate(RepairEvent{Stage: stage, Title: title, Message: message, Output: output, Progress: progress})
	})
	if err != nil {
		a.emitUpdate(RepairEvent{
			Stage: "error", Title: "업데이트 초기화를 완료하지 못했습니다",
			Message: "관련 서비스는 다시 시작했습니다. PC를 재시작한 뒤 다시 시도하거나 빠른 지원을 이용해 주세요.",
			Output:  err.Error(), Progress: 0,
		})
		return
	}
	a.emitUpdate(RepairEvent{
		Stage: "complete", Title: "업데이트 캐시 초기화 완료",
		Message:  "PC를 다시 시작한 다음 Windows 업데이트에서 업데이트 확인을 눌러 새 파일을 내려받으세요.",
		Progress: 100,
	})
}

func (a *App) emit(event RepairEvent) {
	if a.emitEvent != nil {
		a.emitEvent(event)
	}
}

func (a *App) emitUpdate(event RepairEvent) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "update:event", event)
	}
}

func friendlyError(step CommandSpec, err error) string {
	if errors.Is(err, errUnsupportedPlatform) {
		return "이 앱은 Windows 10 및 Windows 11에서만 실행할 수 있습니다."
	}
	return fmt.Sprintf("%s 단계에서 오류가 발생했습니다. 앱을 관리자 권한으로 실행했는지와 인터넷 연결을 확인한 뒤 다시 시도해 주세요.", step.Title)
}

func koreanDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%d분 %d초", minutes, seconds)
	}
	return fmt.Sprintf("%d초", seconds)
}
