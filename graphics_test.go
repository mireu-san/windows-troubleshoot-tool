package main

import (
	"strings"
	"testing"
)

func TestGraphicsVendorUsesHardwareID(t *testing.T) {
	cases := map[string]string{
		`PCI\VEN_8086&DEV_1234\1`: "intel", `pci\ven_1002&dev_9876\2`: "amd", `PCI\VEN_10DE&DEV_4321\3`: "nvidia",
		`ROOT\Intel Display`: "unknown", `PCI\VEN_80860&DEV_1234`: "unknown", `Intel Core i7`: "unknown", `ROOT\RDPIDD\0000`: "unknown",
	}
	for id, want := range cases {
		if got := graphicsVendor(id); got != want {
			t.Errorf("%s: %s != %s", id, got, want)
		}
	}
}
func TestGraphicsFindingsRequireEvidence(t *testing.T) {
	r := &GraphicsReport{GPUs: []GraphicsGPU{{Name: "Intel GPU", ID: `PCI\VEN_8086&DEV_1234`}}}
	analyzeGraphics(r)
	if len(r.Findings) != 1 || r.Findings[0].Title != "관련 오류 근거를 찾지 못했습니다" {
		t.Fatalf("unsupported diagnosis: %+v", r.Findings)
	}
	r.Events = []GraphicsLogEvent{{Provider: "Display", ID: 4101, Time: "2026-10-05T12:00:00+09:00"}, {Provider: "Windows Error Reporting", ID: 1001, Data: map[string]string{"EventName": "LiveKernelEvent", "P1": "141"}}}
	analyzeGraphics(r)
	if len(r.Findings) != 2 || r.Findings[0].Title != "디스플레이 드라이버 복구 기록 발견" || r.Findings[1].Title != "GPU 시간 초과 관련 기록 발견" {
		t.Fatalf("findings: %+v", r.Findings)
	}
}
func TestGraphicsDeviceErrorAndBoundedFindings(t *testing.T) {
	code := uint32(43)
	r := &GraphicsReport{GPUs: []GraphicsGPU{{Name: "GPU", ProblemCode: &code}}}
	for i := 0; i < 300; i++ {
		r.Events = append(r.Events, GraphicsLogEvent{Provider: "Display", ID: 4101})
	}
	analyzeGraphics(r)
	if len(r.Findings) != 30 || !strings.Contains(r.Findings[0].Evidence, "43") {
		t.Fatalf("findings: %d", len(r.Findings))
	}
}
func TestGraphicsUpdateComparisonMatchesDeviceNotArrayOrder(t *testing.T) {
	before := []GraphicsGPU{{ID: "A", Name: "Intel", Version: "1"}, {ID: "B", Name: "NVIDIA", Version: "2"}}
	after := []GraphicsGPU{{ID: "b", Name: "NVIDIA", Version: "3"}, {ID: "a", Name: "Intel", Version: "1"}}
	changes := compareGraphicsVersions(before, after)
	if len(changes) != 1 || changes[0] != "NVIDIA: 2 → 3" {
		t.Fatal(changes)
	}
	if graphicsUpdateURL("intel") != "https://www.intel.com/content/www/us/en/support/detect.html" || graphicsUpdateURL("other") != "" {
		t.Fatal("vendor routing")
	}
}
func TestGraphicsUpdateRequiresDetectedVendor(t *testing.T) {
	app := &App{graphicsReport: &GraphicsReport{GPUs: []GraphicsGPU{{Vendor: "intel"}}}}
	if _, err := app.StartGraphicsUpdate("nvidia"); err == nil {
		t.Fatal("undetected vendor accepted")
	}
	if app.IsRunning() {
		t.Fatal("operation lock retained after rejection")
	}
}
