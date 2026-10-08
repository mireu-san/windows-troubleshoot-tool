package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHardwareNoComponentVerdict(t *testing.T) {
	for _, events := range [][]HardwareEvent{nil, {{Source: "Microsoft-Windows-Kernel-Power", ID: "41", Relevance: "비정상 종료"}}, {{Source: "Display", ID: "4101", Relevance: "그래픽"}}} {
		r := &HardwareReport{Events: events}
		docs := hardwareDocuments(r)
		report := string(docs["hardware-diagnostic-report.md"])
		if !strings.Contains(hardwareConclusion(r), "메인보드") || !strings.Contains(report, "PSU 고장 증명이 아닙니다") || strings.Count(report, "신뢰도: Low") != 9 {
			t.Fatal("unsafe or incomplete hypothesis report")
		}
	}
}
func TestHardwareExportAndCorrelation(t *testing.T) {
	r := &HardwareReport{Request: HardwareRequest{Incident: "2026-10-08T10:00:00+09:00", Symptoms: "<script>bad</script>"}, Sections: map[string]HardwareSection{"CPU": {Status: "수집 완료", Data: json.RawMessage(`[{"Name":"CPU"}]`)}}, Events: []HardwareEvent{{Time: "2026-10-08T01:00:10Z", Source: "=unsafe", Description: "comma, newline\ntext", Relevance: "그래픽"}}}
	root := t.TempDir()
	a, err := writeHardwareReport(root, r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := writeHardwareReport(root, r)
	if err != nil || a == b {
		t.Fatal("folder collision", err)
	}
	entries, _ := os.ReadDir(a)
	if len(entries) != 3 {
		t.Fatal("expected three artifacts")
	}
	raw, _ := os.ReadFile(filepath.Join(a, "hardware-events.csv"))
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff"))).ReadAll()
	if err != nil || len(rows) != 2 {
		t.Fatal(err)
	}
	if rows[1][1] != "'=unsafe" || !strings.Contains(rows[1][5], "+10초") {
		t.Fatal(rows)
	}
	md, _ := os.ReadFile(filepath.Join(a, "hardware-diagnostic-report.md"))
	if strings.Contains(string(md), "<script>") {
		t.Fatal("unescaped user markup")
	}
}
