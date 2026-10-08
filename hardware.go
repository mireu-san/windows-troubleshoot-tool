package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type HardwareRequest struct {
	Incident string `json:"incident"`
	Wiring   string `json:"wiring"`
	Symptoms string `json:"symptoms"`
}
type HardwareSection struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}
type HardwareEvent struct {
	Time        string `json:"time"`
	Source      string `json:"source"`
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Relevance   string `json:"relevance"`
}
type HardwareReport struct {
	Collected string                     `json:"collected"`
	From      string                     `json:"from"`
	Request   HardwareRequest            `json:"request"`
	Sections  map[string]HardwareSection `json:"sections"`
	Events    []HardwareEvent            `json:"events"`
	Issues    []string                   `json:"issues"`
}
type HardwareResult struct {
	Folder  string `json:"folder"`
	Summary string `json:"summary"`
}

func (a *App) InvestigateHardware(request HardwareRequest) (*HardwareResult, error) {
	if len(request.Wiring)+len(request.Symptoms) > 12000 {
		return nil, errors.New("증상 입력이 너무 깁니다")
	}
	now := time.Now()
	if request.Incident != "" {
		incident, err := time.Parse(time.RFC3339, request.Incident)
		if err != nil || incident.After(now) {
			return nil, errors.New("발생 시각을 확인해 주세요")
		}
	}
	if !a.beginOperation() {
		return nil, errors.New("다른 작업이 진행 중입니다")
	}
	defer a.endOperation()
	a.mu.Lock()
	a.hardwareReport = nil // Never retry a stale snapshot after a new collection fails.
	a.mu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Minute)
	defer cancel()
	report, err := collectHardware(ctx, now.AddDate(0, 0, -30), now, request)
	if err != nil {
		return nil, err
	}
	report.Request = request
	report.Collected = now.Format(time.RFC3339)
	report.From = now.AddDate(0, 0, -30).Format(time.RFC3339)
	sort.SliceStable(report.Events, func(i, j int) bool { return report.Events[i].Time < report.Events[j].Time })
	// Keep the snapshot available if saving fails; retry does not repeat collection.
	a.mu.Lock()
	a.hardwareReport = report
	a.mu.Unlock()
	return a.saveHardwareReport(report)
}
func (a *App) SaveHardwareReport() (*HardwareResult, error) {
	if !a.beginOperation() {
		return nil, errors.New("다른 작업이 진행 중입니다")
	}
	defer a.endOperation()
	a.mu.Lock()
	report := a.hardwareReport
	a.mu.Unlock()
	if report == nil {
		return nil, errors.New("먼저 하드웨어 조사를 실행하세요")
	}
	return a.saveHardwareReport(report)
}
func (a *App) saveHardwareReport(report *HardwareReport) (*HardwareResult, error) {
	desktop, err := hardwareDesktop(a.ctx)
	if err != nil {
		return nil, err
	}
	folder, err := writeHardwareReport(desktop, report)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.hardwareFolder = folder
	a.mu.Unlock()
	return &HardwareResult{folder, hardwareConclusion(report)}, nil
}
func (a *App) OpenHardwareFolder() error {
	a.mu.Lock()
	folder := a.hardwareFolder
	a.mu.Unlock()
	if folder == "" {
		return errors.New("저장된 결과가 없습니다")
	}
	return openHardwareFolder(folder)
}
func hardwareConclusion(r *HardwareReport) string {
	for _, e := range r.Events {
		if strings.Contains(e.Relevance, "그래픽") {
			return "그래픽 처리 경로와 관련된 기록이 있습니다. 사건 시각과 대조해야 하며 드라이버·GPU·전원 중 하나를 확정할 수 없습니다. 메인보드 우선 교체를 정당화할 근거는 부족합니다."
		}
	}
	return "수집된 자료만으로 메인보드를 가장 유력한 원인으로 선정할 수 없습니다. 기록이 없다는 것은 정상이라는 뜻이 아닙니다. 부품별 고장 확률 순위는 산정하지 않습니다."
}
func hardwareDocuments(r *HardwareReport) map[string][]byte {
	var inv, md, events strings.Builder
	fmt.Fprintf(&inv, "시스템 구성 — 수집: %s\n", r.Collected)
	keys := make([]string, 0, len(r.Sections))
	for k := range r.Sections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s := r.Sections[k]
		var formatted strings.Builder
		var value interface{}
		_ = json.Unmarshal(s.Data, &value)
		raw, _ := json.MarshalIndent(value, "", "  ")
		formatted.Write(raw)
		fmt.Fprintf(&inv, "\n[%s] %s\n%s\n", k, s.Status, formatted.String())
	}
	missing := "실제 배선·변환기 방향/전원·PSU 모델/출력 안정성·콘덴서 상태는 수동 확인 필요. GPU 이름만으로 내장 여부를 확정하지 않습니다. 모니터 인터페이스는 Windows 보고값이며 실제 변환 경로를 입증하지 않습니다. SMART/건강 상태가 정상이어도 전체 장치의 정상 동작을 보증하지 않습니다. 빈 필드는 확인 불가입니다."
	fmt.Fprintf(&inv, "\n확인 불가 및 제한\n%s\n%s\n", missing, strings.Join(r.Issues, "\n"))
	fmt.Fprintf(&md, "# 하드웨어 진단 보고서\n\n## 요약\n\n%s\n\n수집: %s\n\n기본 조회: %s → %s (입력한 사건 전후 2분은 별도 조회)\n\n## 사용자 진술\n\n발생 시각: %s\n\n배선: %s\n\n증상 및 전원/녹화 지속 여부: %s\n\n## 시스템 사양\n\n```text\n%s\n```\n", hardwareConclusion(r), r.Collected, r.From, r.Collected, mdText(r.Request.Incident), mdText(r.Request.Wiring), mdText(r.Request.Symptoms), strings.ReplaceAll(inv.String(), "```", "'''"))
	md.WriteString("\n## 주요 발견과 사건 타임라인\n\nKernel-Power 41/6008은 비정상 종료의 단서이며 PSU 고장 증명이 아닙니다. WHEA·LiveKernelEvent 117/141은 특정 부품의 고장을 확정하지 않습니다. 시간상 인접한 기록도 인과관계를 입증하지 않습니다.\n\n")
	writer := csv.NewWriter(&events)
	writer.UseCRLF = true
	_ = writer.Write([]string{"Timestamp", "Event source", "Event ID", "Severity", "Description", "Diagnostic relevance"})
	for _, e := range r.Events {
		relevance := e.Relevance
		if incident, err := time.Parse(time.RFC3339, r.Request.Incident); err == nil {
			if stamp, err := time.Parse(time.RFC3339Nano, e.Time); err == nil {
				delta := stamp.Sub(incident)
				if delta >= -2*time.Minute && delta <= 2*time.Minute {
					relevance += fmt.Sprintf(" / 사용자 사건과 %+.0f초 차이", delta.Seconds())
				}
			}
		}
		row := []string{e.Time, e.Source, e.ID, e.Severity, e.Description, relevance}
		for i := range row {
			row[i] = csvSafe(row[i])
		}
		_ = writer.Write(row)
		fmt.Fprintf(&md, "- %s · %s · ID %s · %s — %s\n", mdText(e.Time), mdText(e.Source), mdText(e.ID), mdText(relevance), mdText(e.Description))
	}
	writer.Flush()
	if len(r.Events) == 0 {
		md.WriteString("관련 기록을 확보하지 못했습니다. 조회 상태와 보존 기간을 확인하세요.\n")
	}
	md.WriteString("\n## 가설별 평가\n\n신뢰도는 해당 부품을 원인으로 특정하는 신뢰도입니다. 자동 수집만으로 높은 신뢰도를 부여하지 않습니다.\n\n")
	hypotheses := [][3]string{
		{"메인보드 전원부·출력 회로", "하드웨어", "정상 케이블·변환기 우회 후에도 여러 출력에서 재현되는지 비교"},
		{"Intel 내장 GPU·CPU", "그래픽", "실제 CPU/GPU 모델과 WHEA 세부 장치 확인 후 기술자 교차 검사"},
		{"PSU 전원 불안정", "비정상 종료", "저비용 연결 검사 이후 필요하면 기술자가 정상 PSU로 교차 검사"},
		{"VGA–HDMI 변환기", "", "변환기가 있다면 동일 모니터를 정상 디지털 케이블로 직접 연결"},
		{"케이블·커넥터", "", "같은 포트·모니터에서 케이블만 정상 제품으로 교체 비교"},
		{"모니터 자체", "", "같은 모니터를 정상 확인된 다른 영상 소스에 연결 비교"},
		{"그래픽 드라이버", "그래픽", "4101/117/141 시각과 증상 대조; 변경 검사는 별도 승인 후"},
		{"RAM 불안정", "하드웨어", "DIMM 구성·WHEA 확인; 녹화 중단 가능한 메모리 검사는 별도 승인 후"},
		{"Windows·펌웨어", "그래픽", "BIOS/Windows 버전과 오류 시각 확인; 변경은 별도 승인 후"},
	}
	for _, h := range hypotheses {
		refs := []string{}
		if h[1] != "" {
			for _, e := range r.Events {
				if strings.Contains(e.Relevance, h[1]) && len(refs) < 5 {
					refs = append(refs, e.Time+" "+e.Source+" ID "+e.ID)
				}
			}
		}
		support := "직접 지지하는 자동 수집 증거 없음"
		if len(refs) > 0 {
			support = "관련 단서(부품 특정 증거 아님): " + strings.Join(refs, "; ")
		}
		fmt.Fprintf(&md, "### %s\n\n- 지지 증거: %s\n- 반대 증거: 원인을 배제할 비교 검사 자료 없음. 로그 부재는 반대 증거로 사용하지 않음.\n- 부족한 증거: 재현 조건, 사건과의 일치, 해당 부품의 교차 검사 결과\n- 다음 검사: %s\n- 신뢰도: Low (낮음)\n\n", h[0], support, h[2])
	}
	md.WriteString("## 다음 검사와 교체 판단\n\n1. 설정을 바꾸지 않고 발생 시각·모니터 신호 없음 표시·PC 조작 및 녹화 지속 여부를 관찰합니다. 녹화 파일은 이 도구가 열지 않습니다.\n2. 가장 정보가 많은 다음 물리 검사: 변환기를 사용 중이고 직접 연결이 가능하면, 문제가 나타나는 동일 모니터를 정상 확인된 디지털 케이블로 PC에 직접 연결해 기존 변환 경로와 비교합니다. 변환기가 없다면 케이블 하나만 바꿔 비교합니다. 이는 변환기와 케이블을 포함한 경로 검사이며 개별 부품 확정은 아닙니다.\n3. 승인된 작업 시간에 모니터를 한 대씩 비교하고, 이후 케이블·모니터·포트를 한 변수씩 바꿉니다.\n4. 계속 재현되면 기술자의 정상 PSU 교차 검사, 이후 RAM/CPU/메인보드 검사를 검토합니다. 녹화에 영향을 줄 수 있는 분리·재시작·부하 검사는 별도 승인 전 실행하지 않습니다.\n\n메인보드 교체: 현재 자동 수집만으로 우선 교체는 정당화되지 않습니다. 저렴한 경로 검사와 전원 교차 검사를 먼저 검토하세요.\n\nCPU·RAM 재사용: 실제 CPU 모델·소켓·보드 CPU 지원 목록/BIOS 및 메모리 규격을 확인해야 합니다. DDR4 또는 정상 상태라고 추정하지 않으며, 호환성과 건강 상태를 별도로 검증해야 합니다.\n\n## 수집 범위와 개인정보\n\n설정 변경·설치·복구·재부팅·부하 검사 없이 조회했습니다. CCTV·거래/브라우저 자료는 수집하지 않았습니다. 원본 이벤트 자유문·사용자/PC 이름·전체 파일 경로·장치 일련번호 대신 허용된 진단 필드와 요약을 내보냅니다. 사용자 입력에는 개인정보를 기재하지 마세요. WER/덤프는 지정 진단 폴더만 조회하며 덤프 내용은 읽거나 복사하지 않습니다. 수집 자체의 CPU/디스크 사용량은 0이 아닙니다.\n")
	return map[string][]byte{"hardware-diagnostic-report.md": []byte(md.String()), "system-inventory.txt": append([]byte{0xef, 0xbb, 0xbf}, []byte(inv.String())...), "hardware-events.csv": append([]byte{0xef, 0xbb, 0xbf}, []byte(events.String())...)}
}
func mdText(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", "<", "&lt;", ">", "&gt;", "`", "'", "[", "(", "]", ")", "*", "", "#", "").Replace(s)
}
func csvSafe(s string) string {
	if strings.HasPrefix(strings.TrimSpace(s), "=") || strings.HasPrefix(strings.TrimSpace(s), "+") || strings.HasPrefix(strings.TrimSpace(s), "-") || strings.HasPrefix(strings.TrimSpace(s), "@") {
		return "'" + s
	}
	return s
}
func writeHardwareReport(desktop string, r *HardwareReport) (string, error) {
	if desktop == "" {
		return "", errors.New("Windows 바탕화면 경로를 확인하지 못했습니다")
	}
	folder, err := os.MkdirTemp(desktop, "하드웨어진단_"+time.Now().Format("2006-01-02_15-04-05")+"_")
	if err != nil {
		return "", err
	}
	for name, data := range hardwareDocuments(r) {
		if err = os.WriteFile(filepath.Join(folder, name), data, 0600); err != nil {
			return "", fmt.Errorf("일부 저장 실패 (%s). 다시 저장을 사용하세요: %w", folder, err)
		}
	}
	return folder, nil
}
