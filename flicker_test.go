package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestFlickerSeparatesCPUAndSecurityEvidence(t *testing.T) {
	r := &GraphicsReport{Events: []GraphicsLogEvent{
		{Provider: "Microsoft-Windows-WHEA-Logger", ID: 17, Data: map[string]string{"ApicId": "0"}},
		{Provider: "Display", ID: 4101},
	}, Comprehensive: &FlickerDetails{Firewall: []FirewallProfile{{Name: "Public", Enabled: "False"}, {Name: "Private", Enabled: "True"}, {Name: "Domain", Enabled: "NotConfigured"}}}}
	analyzeGraphics(r)
	analyzeFlicker(r)
	titles := func() string {
		var s strings.Builder
		for _, f := range r.Findings {
			s.WriteString(f.Title + "\n")
		}
		return s.String()
	}
	if strings.Contains(titles(), "CPU 관련 하드웨어 오류 기록") || !strings.Contains(titles(), "CPU 원인 판단 근거 부족") {
		t.Fatal("PCIe WHEA must not imply CPU failure", r.Findings)
	}
	if strings.Count(titles(), "방화벽 비활성 프로필 발견") != 1 {
		t.Fatal(r.Findings)
	}
	r.Events = append(r.Events, GraphicsLogEvent{Provider: "Microsoft-Windows-WHEA-Logger", ID: 18, Data: map[string]string{"ApicId": "2"}})
	analyzeGraphics(r)
	analyzeFlicker(r)
	if !strings.Contains(titles(), "CPU 관련 하드웨어 오류 기록") || !strings.Contains(titles(), "디스플레이 드라이버 복구 기록 발견") {
		t.Fatal("mixed evidence lost")
	}
}

func TestFlickerUnknownSecurityDoesNotMeanDisabled(t *testing.T) {
	r := &GraphicsReport{Comprehensive: &FlickerDetails{Issues: []string{"Access denied"}}}
	analyzeFlicker(r)
	for _, f := range r.Findings {
		if strings.Contains(f.Title, "비활성") {
			t.Fatal("missing values treated as disabled")
		}
	}
}

func TestFlickerTextPreservesEvidenceAndEncoding(t *testing.T) {
	r := &GraphicsReport{Collected: "2026-10-06", From: "start", Until: "end", GPUs: []GraphicsGPU{{Name: "그래픽"}}, Comprehensive: &FlickerDetails{Issues: []string{"Security access denied"}}, Events: []GraphicsLogEvent{{Message: "raw event"}}}
	watch := DisplayWatch{Changes: []DisplayChange{{Before: DisplaySample{Monitors: []GraphicsMonitor{{Width: 1920}}}, After: DisplaySample{Monitors: []GraphicsMonitor{{Width: 1280}}, Process: "game.exe"}}}, Markers: []string{"symptom-time"}}
	data := flickerText(r, `{"timing":"separate"}`, watch)
	if !bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatal("missing UTF-8 BOM")
	}
	for _, want := range []string{"그래픽", "raw event", "Security access denied", "1920", "1280", "game.exe", "symptom-time", "separate"} {
		if !strings.Contains(string(data), want) {
			t.Fatal("missing", want)
		}
	}
	if strings.Contains(strings.ReplaceAll(string(data), "\r\n", ""), "\n") {
		t.Fatal("expected Windows newlines")
	}
}

func TestDisplayWatchSnapshotsAreIndependent(t *testing.T) {
	original := DisplayWatch{Running: true, Initial: &DisplaySample{Monitors: []GraphicsMonitor{{Name: "left"}}}, Markers: []string{"one"}, Changes: []DisplayChange{{After: DisplaySample{Monitors: []GraphicsMonitor{{Name: "right"}}}}}}
	copied := cloneWatch(original)
	copied.Initial.Monitors[0].Name = "changed"
	copied.Markers[0] = "changed"
	copied.Changes[0].After.Monitors[0].Name = "changed"
	if original.Initial.Monitors[0].Name != "left" || original.Markers[0] != "one" || original.Changes[0].After.Monitors[0].Name != "right" {
		t.Fatal("snapshot aliases live state")
	}
	a := &App{displayWatch: original}
	if a.StopDisplayWatch().Running || a.GetDisplayWatch().Ended == "" {
		t.Fatal("watch not stopped")
	}
	if _, err := a.ExportFlickerReport("invalid"); err == nil {
		t.Fatal("invalid answers accepted")
	}
}

func TestDisplayWatchCancellationCannotStopNewSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := &App{displayWatch: DisplayWatch{Running: true, Started: "new"}}
	a.watchDisplays(ctx, "old")
	if !a.GetDisplayWatch().Running {
		t.Fatal("old collector stopped new session")
	}
	a.watchDisplays(ctx, "new")
	if a.GetDisplayWatch().Running || a.GetDisplayWatch().Ended == "" {
		t.Fatal("cancelled collector still running")
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	a.displayWatch = DisplayWatch{Running: true, Started: "deadline"}
	a.watchDisplays(ctx, "deadline")
	if a.GetDisplayWatch().Running || len(a.GetDisplayWatch().Issues) != 1 {
		t.Fatal("deadline was not reported")
	}
}
