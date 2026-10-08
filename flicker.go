package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type FirewallProfile struct {
	Name    string `json:"name"`
	Enabled string `json:"enabled"`
}
type FlickerDetails struct {
	CPU             []map[string]interface{} `json:"cpu"`
	Windows         []map[string]interface{} `json:"windows"`
	Firewall        []FirewallProfile        `json:"firewall"`
	Network         []map[string]interface{} `json:"network"`
	Defender        []map[string]interface{} `json:"defender"`
	Antivirus       []map[string]interface{} `json:"antivirus"`
	RemoteProcesses []map[string]interface{} `json:"remoteProcesses"`
	Startup         []map[string]interface{} `json:"startup"`
	Tasks           []map[string]interface{} `json:"tasks"`
	RemoteEvents    []GraphicsLogEvent       `json:"remoteEvents"`
	DriverEvents    []GraphicsLogEvent       `json:"driverEvents"`
	ProcessEvents   []GraphicsLogEvent       `json:"processEvents"`
	Issues          []string                 `json:"issues"`
}

type DisplaySample struct {
	Time       string            `json:"time"`
	Monitors   []GraphicsMonitor `json:"monitors"`
	PID        uint32            `json:"pid"`
	Process    string            `json:"process"`
	Fullscreen *bool             `json:"fullscreen"`
	Issue      string            `json:"issue,omitempty"`
}
type DisplayChange struct {
	Before DisplaySample `json:"before"`
	After  DisplaySample `json:"after"`
}
type DisplayWatch struct {
	Running bool            `json:"running"`
	Started string          `json:"started"`
	Ended   string          `json:"ended"`
	Initial *DisplaySample  `json:"initial,omitempty"`
	Last    *DisplaySample  `json:"last,omitempty"`
	Changes []DisplayChange `json:"changes"`
	Markers []string        `json:"markers"`
	Issues  []string        `json:"issues"`
}

func (a *App) ScanFlickerDiagnostics() (*GraphicsReport, error) { return a.scanGraphics(false, true) }

func analyzeFlicker(r *GraphicsReport) {
	add := func(title, evidence, action string) {
		r.Findings = append(r.Findings, GraphicsFinding{title, evidence, action})
	}
	cpu := false
	for _, e := range r.Events {
		// WHEA processor machine-check records are CPU-related evidence, not a CPU failure verdict.
		if e.Provider == "Microsoft-Windows-WHEA-Logger" && (e.ID == 18 || e.ID == 19) && e.Data["ApicId"] != "" {
			cpu = true
			add("CPU 관련 하드웨어 오류 기록", fmt.Sprintf("%s · ID %d · APIC %s", e.Time, e.ID, e.Data["ApicId"]), "프로세서가 보고한 오류입니다. CPU·메모리·전원·펌웨어 등을 함께 점검해야 하며 CPU 고장을 확정하지 않습니다.")
		}
	}
	if !cpu {
		add("CPU 원인 판단 근거 부족", "CPU 관련 오류를 명확히 식별하지 못했습니다.", "오류 기록이 없어도 정상이라고 확정할 수 없습니다.")
	}
	add("내장 그래픽 원인 확인 필요", "GPU 제조사만으로 내장·외장 여부나 오류 원인을 확정하지 않습니다.", "오류 이벤트의 장치와 모니터 연결 GPU를 비교하세요. 복구 전후 드라이버 변경과 앱의 화면 모드 전환도 확인하세요.")
	if r.Comprehensive == nil {
		add("종합 정보 수집 실패", "CPU·보안 정보가 완전히 수집되지 않았습니다.", "수집 제한을 확인하고 다시 검사하세요.")
		return
	}
	d := r.Comprehensive
	for _, p := range d.Firewall {
		if strings.EqualFold(p.Enabled, "False") || p.Enabled == "0" {
			add("방화벽 비활성 프로필 발견", p.Name+": "+p.Enabled, "현재 네트워크 프로필과 비교하세요. 보안 설정 취약점이며 침입이나 깜박임 원인의 증거는 아닙니다. 관리 정책과 타사 보안 제품을 확인하세요.")
		}
	}
	for _, state := range d.Defender {
		if enabled, ok := state["RealTimeProtectionEnabled"].(bool); ok && !enabled {
			add("Defender 실시간 보호 비활성", fmt.Sprint(state["AMRunningMode"]), "타사 백신 및 관리 정책을 확인하세요. 이 정보만으로 전체 백신 보호 상태나 감염 여부를 확정할 수 없습니다.")
		}
	}
	if len(d.RemoteProcesses) > 0 || len(d.RemoteEvents) > 0 {
		add("원격 접속 관련 단서", fmt.Sprintf("알려진 원격 도구 프로세스 %d개, 세션 이벤트 %d개", len(d.RemoteProcesses), len(d.RemoteEvents)), "승인된 원격 작업인지 확인하고 발생 시각을 비교하세요. 정상 관리 작업일 수 있으며 침입을 의미하지 않습니다.")
	}
	add("듀얼 모니터 추가 점검", "자동 수집만으로 케이블·포트·모니터·전원 문제를 판별할 수 없습니다.", "디스플레이 변경 감시를 켜고 증상을 재현하세요. 모니터별 주사율, HDR/VRR, 앱 하드웨어 가속을 확인하고 케이블·포트를 교차 점검하세요.")
}

func canonicalSample(s DisplaySample) DisplaySample {
	sort.Slice(s.Monitors, func(i, j int) bool {
		return s.Monitors[i].Device+s.Monitors[i].ID+s.Monitors[i].Name < s.Monitors[j].Device+s.Monitors[j].ID+s.Monitors[j].Name
	})
	return s
}
func (a *App) StartDisplayWatch() (DisplayWatch, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.displayWatch.Running {
		return cloneWatch(a.displayWatch), nil
	}
	sample, err := captureDisplaySample()
	if err != nil {
		return DisplayWatch{}, err
	}
	sample = canonicalSample(sample)
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
	a.displayWatchCancel = cancel
	a.displayWatch = DisplayWatch{Running: true, Started: sample.Time, Initial: &sample, Last: &sample, Changes: []DisplayChange{}, Markers: []string{}, Issues: []string{}}
	go a.watchDisplays(ctx, sample.Time)
	return cloneWatch(a.displayWatch), nil
}
func cloneWatch(w DisplayWatch) DisplayWatch {
	// Copies protect returned bridge/export values while the collector continues.
	data, _ := json.Marshal(w)
	var result DisplayWatch
	_ = json.Unmarshal(data, &result)
	return result
}
func (a *App) watchDisplays(ctx context.Context, started string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer func() {
		a.mu.Lock()
		defer a.mu.Unlock()
		// Never finish a newer session when a cancelled collector exits late.
		if a.displayWatch.Started != started || !a.displayWatch.Running {
			return
		}
		a.displayWatch.Running = false
		a.displayWatch.Ended = time.Now().Format(time.RFC3339Nano)
		if ctx.Err() == context.DeadlineExceeded {
			a.displayWatch.Issues = append(a.displayWatch.Issues, "30 minute monitoring limit reached")
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sample, err := captureDisplaySample()
			a.mu.Lock()
			if ctx.Err() != nil || !a.displayWatch.Running || a.displayWatch.Started != started {
				a.mu.Unlock()
				return
			}
			if err != nil {
				a.displayWatch.Issues = append(a.displayWatch.Issues, err.Error())
				a.displayWatch.Running = false
				a.displayWatch.Ended = time.Now().Format(time.RFC3339Nano)
				a.displayWatchCancel()
				a.mu.Unlock()
				return
			}
			sample = canonicalSample(sample)
			last := a.displayWatch.Last
			if last != nil && !reflect.DeepEqual(last.Monitors, sample.Monitors) {
				a.displayWatch.Changes = append(a.displayWatch.Changes, DisplayChange{*last, sample})
			}
			a.displayWatch.Last = &sample
			if len(a.displayWatch.Changes) >= 500 {
				a.displayWatch.Issues = append(a.displayWatch.Issues, "500 display changes reached; monitoring stopped")
				a.displayWatch.Running = false
				a.displayWatch.Ended = sample.Time
				a.displayWatchCancel()
			}
			a.mu.Unlock()
		}
	}
}
func (a *App) GetDisplayWatch() DisplayWatch {
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneWatch(a.displayWatch)
}
func (a *App) StopDisplayWatch() DisplayWatch {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.displayWatch.Running {
		a.displayWatch.Running = false
		a.displayWatch.Ended = time.Now().Format(time.RFC3339Nano)
		if a.displayWatchCancel != nil {
			a.displayWatchCancel()
		}
	}
	return cloneWatch(a.displayWatch)
}

func flickerText(report *GraphicsReport, answers string, watch DisplayWatch) []byte {
	var b strings.Builder
	b.WriteString("화면 깜박임 종합 진단 / Flicker diagnostics\n")
	fmt.Fprintf(&b, "수집 시각: %s\n검사 기간: %s → %s\n깜박임 표시: %s\n\n사용자 증상 답변 (사용자 진술):\n%s\n\n진단 요약:\n", report.Collected, report.From, report.Until, report.Marker, answers)
	fmt.Fprintf(&b, "PC: %s %s\n드라이버 비교 기준 시각: %s\nWindows 재부팅 대기: %t\n", report.Manufacturer, report.Model, report.BaselineTime, report.RestartPending)
	for _, f := range report.Findings {
		fmt.Fprintf(&b, "\n- %s\n  근거: %s\n  다음 확인: %s\n", f.Title, f.Evidence, f.Action)
	}
	b.WriteString("\n해석 및 수집 제한:\n1초 간격 표본이므로 짧은 변경은 놓칠 수 있습니다. 활성 앱과 전체 화면 추정은 변경을 일으킨 앱의 증거가 아닙니다. 감시 시작 전 변경은 기록되지 않습니다. 다시 시작하면 이전 감시 기록이 대체됩니다.\n좌표는 Windows 가상 화면 배치이며 실제 좌우 위치와 다를 수 있습니다. GPU 내장/외장 여부, HDR/VRR, 케이블·포트·전원은 별도 확인이 필요합니다.\n방화벽 비활성화는 침입 또는 화면 증상의 원인으로 단정할 수 없습니다. 이벤트 부재도 정상이나 안전함을 입증하지 않습니다.\n예약 작업·시작 항목은 검토용 목록이며 악성 여부를 판정하지 않습니다. 로그와 프로세스 정보에 사용자명·경로·주소가 포함될 수 있습니다.\n")
	sections := []struct {
		name  string
		value interface{}
	}{
		{"CPU·Windows·보안·원격 접속·드라이버 변경 자료", report.Comprehensive},
		{"GPU 장치와 드라이버", report.GPUs}, {"모니터별 좌표·연결·화면 모드", report.Monitors},
		{"관련 이벤트 원문", report.Events}, {"수집 실패·제한", report.Issues},
		{"드라이버 기준 상태와 비교", report.Changes}, {"디스플레이 변경 전후·활성 앱·깜박임 기록", watch},
	}
	for _, section := range sections {
		raw, _ := json.MarshalIndent(section.value, "", "  ")
		fmt.Fprintf(&b, "\n[%s]\n%s\n", section.name, raw)
	}
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(strings.ReplaceAll(b.String(), "\n", "\r\n"))...)
}
func (a *App) ExportFlickerReport(answers string) (string, error) {
	if len(answers) > 16000 || !json.Valid([]byte(answers)) {
		return "", errors.New("진단 답변 형식 또는 길이를 확인해 주세요.")
	}
	a.mu.Lock()
	report := a.flickerReport
	watch := cloneWatch(a.displayWatch)
	a.mu.Unlock()
	if report == nil {
		return "", errors.New("먼저 그래픽 검사를 실행해 주세요.")
	}
	data := flickerText(report, answers, watch)
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: translateNative("깜박임 원인 종합 진단 저장", a.GetLanguageSettings().Language), DefaultFilename: "flicker-diagnostics-" + time.Now().Format("20060102-150405") + ".txt", Filters: []runtime.FileFilter{{DisplayName: "Text (*.txt)", Pattern: "*.txt"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0600)
}
