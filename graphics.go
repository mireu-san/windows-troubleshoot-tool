package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type GraphicsGPU struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Vendor      string  `json:"vendor"`
	Version     string  `json:"version"`
	Provider    string  `json:"provider"`
	DriverDate  string  `json:"driverDate"`
	INF         string  `json:"inf"`
	Signed      *bool   `json:"signed"`
	ProblemCode *uint32 `json:"problemCode"`
	Status      string  `json:"status"`
}
type GraphicsMonitor struct {
	ID        string `json:"id"`
	X         int32  `json:"x"`
	Y         int32  `json:"y"`
	Primary   bool   `json:"primary"`
	ModeKnown bool   `json:"modeKnown"`
	Name      string `json:"name"`
	Adapter   string `json:"adapter"`
	AdapterID string `json:"adapterId"`
	Device    string `json:"device"`
	Width     uint32 `json:"width"`
	Height    uint32 `json:"height"`
	Hz        uint32 `json:"hz"`
}
type GraphicsLogEvent struct {
	Time     string            `json:"time"`
	Provider string            `json:"provider"`
	ID       uint32            `json:"id"`
	RecordID uint64            `json:"recordId"`
	Log      string            `json:"log"`
	Message  string            `json:"message"`
	Data     map[string]string `json:"data"`
}
type GraphicsFinding struct {
	Title    string `json:"title"`
	Evidence string `json:"evidence"`
	Action   string `json:"action"`
}
type GraphicsReport struct {
	Comprehensive  *FlickerDetails    `json:"comprehensive,omitempty"`
	Collected      string             `json:"collected"`
	From           string             `json:"from"`
	Until          string             `json:"until"`
	Marker         string             `json:"marker,omitempty"`
	Manufacturer   string             `json:"manufacturer"`
	Model          string             `json:"model"`
	GPUs           []GraphicsGPU      `json:"gpus"`
	Monitors       []GraphicsMonitor  `json:"monitors"`
	Events         []GraphicsLogEvent `json:"events"`
	Issues         []string           `json:"issues"`
	Findings       []GraphicsFinding  `json:"findings"`
	Changes        []string           `json:"changes"`
	RestartPending bool               `json:"restartPending"`
	BaselineTime   string             `json:"baselineTime,omitempty"`
}
type GraphicsUpdateResult struct {
	Vendor   string `json:"vendor"`
	Launched bool   `json:"launched"`
	URL      string `json:"url"`
	Message  string `json:"message"`
}
type graphicsBaseline struct {
	Time string        `json:"time"`
	GPUs []GraphicsGPU `json:"gpus"`
}

func graphicsVendor(id string) string {
	id = strings.ToUpper(id)
	if !strings.HasPrefix(id, "PCI\\") {
		return "unknown"
	}
	for token, vendor := range map[string]string{"VEN_8086": "intel", "VEN_1002": "amd", "VEN_10DE": "nvidia"} {
		for _, part := range strings.Split(strings.SplitN(strings.TrimPrefix(id, "PCI\\"), "\\", 2)[0], "&") {
			if part == token {
				return vendor
			}
		}
	}
	return "unknown"
}
func graphicsUpdateURL(vendor string) string {
	switch vendor {
	case "intel":
		return "https://www.intel.com/content/www/us/en/support/detect.html"
	case "amd":
		return "https://www.amd.com/en/resources/support-articles/faqs/GPU-131.html"
	case "nvidia":
		return "https://www.nvidia.com/en-us/software/nvidia-app/"
	}
	return ""
}
func analyzeGraphics(report *GraphicsReport) {
	report.Findings = []GraphicsFinding{}
	for i := range report.GPUs {
		gpu := &report.GPUs[i]
		gpu.Vendor = graphicsVendor(gpu.ID)
		if gpu.ProblemCode != nil && *gpu.ProblemCode != 0 {
			report.Findings = append(report.Findings, GraphicsFinding{"그래픽 장치 오류가 보고되었습니다", fmt.Sprintf("%s: Device Manager code %d", gpu.Name, *gpu.ProblemCode), "장치 관리자에서 해당 장치 상태를 확인하고 드라이버 업데이트를 진행하세요."})
		}
		if strings.Contains(strings.ToLower(gpu.Name), "microsoft basic display") {
			report.Findings = append(report.Findings, GraphicsFinding{"기본 디스플레이 드라이버 사용 중", gpu.Name, "감지된 GPU 제조사의 드라이버 또는 PC 제조사 드라이버를 확인하세요."})
		}
	}
	for _, e := range report.Events {
		title, action := "", ""
		switch {
		case e.Provider == "Display" && e.ID == 4101:
			title = "디스플레이 드라이버 복구 기록 발견"
			action = "발생 시각과 깜박임 시각을 비교하고 관련 드라이버를 확인하세요. 이 기록만으로 고장 원인을 확정할 수 없습니다."
		case e.Data["EventName"] == "LiveKernelEvent" && (e.Data["P1"] == "117" || e.Data["P1"] == "141"):
			title = "GPU 시간 초과 관련 기록 발견"
			action = "드라이버와 하드웨어 양쪽을 점검할 단서입니다. 제조사 드라이버와 증상 발생 조건을 확인하세요."
		case strings.EqualFold(e.Data["AppName"], "dwm.exe") || strings.EqualFold(e.Data["AppName"], "explorer.exe"):
			title = "화면 구성 프로그램 오류 기록 발견"
			action = "특정 앱 실행 시 발생하는지 확인하고 앱 업데이트와 하드웨어 가속 설정을 점검하세요."
		case e.Provider == "Microsoft-Windows-WHEA-Logger":
			title = "시스템 하드웨어 오류 기록 발견"
			action = "이 기록은 GPU 고장을 뜻하지 않습니다. 이벤트 세부 정보에서 관련 장치를 확인하세요."
		case e.Log == "Microsoft-Windows-Kernel-PnP/Configuration":
			title = "그래픽 장치 구성 기록 발견"
			action = "드라이버 변경 시각과 증상 시작 시점을 비교하세요. DriverDate는 설치 시각이 아닙니다."
		default:
			title = "그래픽 관련 이벤트 발견"
			action = "이벤트 원문과 증상 발생 시각을 함께 확인하세요."
		}
		if len(report.Findings) < 30 {
			report.Findings = append(report.Findings, GraphicsFinding{title, fmt.Sprintf("%s · %s · ID %d", e.Time, e.Provider, e.ID), action})
		}
	}
	if len(report.Findings) == 0 {
		report.Findings = append(report.Findings, GraphicsFinding{"관련 오류 근거를 찾지 못했습니다", "", "로그가 없어도 정상이라고 확정할 수 없습니다. 증상 질문과 케이블·모니터 점검을 함께 진행하세요."})
	}
}

func (a *App) ScanGraphics() (*GraphicsReport, error)        { return a.scanGraphics(false, false) }
func (a *App) MarkGraphicsFlicker() (*GraphicsReport, error) { return a.scanGraphics(true, false) }
func (a *App) scanGraphics(mark, comprehensive bool) (*GraphicsReport, error) {
	if !a.beginOperation() {
		return nil, errors.New("다른 작업이 진행 중입니다. 완료 후 다시 시도해 주세요.")
	}
	defer a.endOperation()
	now := time.Now()
	a.mu.Lock()
	if mark {
		a.graphicsMarker = now
		if a.displayWatch.Running && len(a.displayWatch.Markers) < 500 {
			a.displayWatch.Markers = append(a.displayWatch.Markers, now.Format(time.RFC3339Nano))
		}
	}
	marker := a.graphicsMarker
	a.mu.Unlock()
	from, until := now.Add(-7*24*time.Hour), now
	// Re-scans include up to two minutes after a recently marked symptom.
	if !comprehensive && !marker.IsZero() && now.Sub(marker) < 10*time.Minute {
		from = marker.Add(-2 * time.Minute)
		if until.After(marker.Add(2 * time.Minute)) {
			until = marker.Add(2 * time.Minute)
		}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 120*time.Second)
	defer cancel()
	report, err := scanGraphicsSystem(ctx, from, until)
	if err != nil {
		return nil, err
	}
	report.Collected = now.Format(time.RFC3339)
	report.From = from.Format(time.RFC3339)
	report.Until = until.Format(time.RFC3339)
	if !marker.IsZero() && now.Sub(marker) < 10*time.Minute {
		report.Marker = marker.Format(time.RFC3339)
	}
	if comprehensive {
		details, detailErr := collectFlickerDetails(ctx, from, until)
		if detailErr != nil {
			report.Issues = append(report.Issues, detailErr.Error())
		} else {
			report.Comprehensive = details
		}
	}
	analyzeGraphics(report)
	if comprehensive {
		analyzeFlicker(report)
	}
	if baseline, err := readGraphicsBaseline(); err == nil {
		report.BaselineTime = baseline.Time
		report.Changes = compareGraphicsVersions(baseline.GPUs, report.GPUs)
	} else if !os.IsNotExist(err) {
		report.Issues = append(report.Issues, "Update baseline: "+err.Error())
	}
	a.mu.Lock()
	a.graphicsReport = report
	if comprehensive {
		a.flickerReport = report
	}
	a.mu.Unlock()
	return report, nil
}
func compareGraphicsVersions(before, after []GraphicsGPU) []string {
	changes := []string{}
	for _, old := range before {
		found := false
		for _, current := range after {
			if strings.EqualFold(old.ID, current.ID) && old.ID != "" {
				found = true
				if old.Version != current.Version {
					changes = append(changes, fmt.Sprintf("%s: %s → %s", current.Name, old.Version, current.Version))
				}
			}
		}
		if !found {
			changes = append(changes, old.Name+": device not found in current scan")
		}
	}
	return changes
}
func graphicsBaselinePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "windows-system-repair-helper", "graphics-baseline.json"), nil
}
func readGraphicsBaseline() (graphicsBaseline, error) {
	var b graphicsBaseline
	path, err := graphicsBaselinePath()
	if err != nil {
		return b, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(data, &b)
	return b, err
}
func saveGraphicsBaseline(gpus []GraphicsGPU) error {
	path, err := graphicsBaselinePath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(graphicsBaseline{time.Now().Format(time.RFC3339), gpus})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
func (a *App) StartGraphicsUpdate(vendor string) (*GraphicsUpdateResult, error) {
	if !a.beginOperation() {
		return nil, errors.New("다른 작업이 진행 중입니다. 완료 후 다시 시도해 주세요.")
	}
	defer a.endOperation()
	a.mu.Lock()
	report := a.graphicsReport
	a.mu.Unlock()
	supported := false
	if report != nil {
		for _, gpu := range report.GPUs {
			if gpu.Vendor == vendor && graphicsUpdateURL(vendor) != "" {
				supported = true
			}
		}
	}
	if !supported {
		return nil, errors.New("먼저 그래픽 검사를 실행하고 감지된 제조사를 선택해 주세요.")
	}
	if err := saveGraphicsBaseline(report.GPUs); err != nil {
		return nil, fmt.Errorf("업데이트 전 상태 저장 실패: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	launched, err := launchGraphicsUpdater(ctx, vendor)
	if err != nil {
		return nil, err
	}
	result := &GraphicsUpdateResult{Vendor: vendor, Launched: launched, URL: graphicsUpdateURL(vendor)}
	if launched {
		result.Message = "제조사 업데이트 도구를 실행했습니다. 도구에서 설치를 마친 뒤 다시 검사해 주세요."
	} else {
		runtime.BrowserOpenURL(a.ctx, result.URL)
		result.Message = "공식 업데이트 페이지를 열었습니다. 도구 설치 또는 드라이버 업데이트를 진행한 뒤 다시 검사해 주세요."
	}
	return result, nil
}
func (a *App) ExportGraphicsReport(answers string) (string, error) {
	if len(answers) > 16000 {
		return "", errors.New("진단 답변이 너무 깁니다")
	}
	a.mu.Lock()
	report := a.graphicsReport
	a.mu.Unlock()
	if report == nil {
		return "", errors.New("먼저 그래픽 검사를 실행해 주세요.")
	}
	data, err := json.MarshalIndent(struct {
		Report  *GraphicsReport `json:"report"`
		Answers string          `json:"symptomAnswers"`
	}{report, answers}, "", "  ")
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: translateNative("그래픽 진단 보고서 저장", a.GetLanguageSettings().Language), DefaultFilename: "graphics-diagnostics.json", Filters: []runtime.FileFilter{{DisplayName: "JSON (*.json)", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0600)
}
func (a *App) OpenGraphicsSettings() error { return openGraphicsSettings() }
func (a *App) OpenDeviceManager() error    { return openDeviceManager() }

func (a *App) OpenPCDriverSupport() (string, error) {
	a.mu.Lock()
	report := a.graphicsReport
	a.mu.Unlock()
	if report == nil {
		return "", errors.New("먼저 그래픽 검사를 실행해 주세요.")
	}
	maker := strings.ToLower(report.Manufacturer)
	var url string
	for name, target := range map[string]string{
		"dell":           "https://www.dell.com/support/home/",
		"lenovo":         "https://pcsupport.lenovo.com/",
		"hp":             "https://support.hp.com/drivers",
		"hewlett":        "https://support.hp.com/drivers",
		"asus":           "https://www.asus.com/support/download-center/",
		"acer":           "https://www.acer.com/support/drivers-and-manuals",
		"microsoft":      "https://support.microsoft.com/surface",
		"samsung":        "https://www.samsung.com/support/",
		"lg electronics": "https://www.lg.com/support",
		"micro-star":     "https://www.msi.com/support",
	} {
		if strings.Contains(maker, name) {
			url = target
			break
		}
	}
	if url == "" {
		return "PC 제조사 지원 사이트에서 위에 표시된 모델의 그래픽 드라이버를 확인해 주세요.", nil
	}
	runtime.BrowserOpenURL(a.ctx, url)
	return "PC 제조사 지원 페이지에서 해당 모델의 그래픽 드라이버를 확인해 주세요.", nil
}
