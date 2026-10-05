package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiagnosticWindowExcludesOtherDatesAndPreservesContext(t *testing.T) {
	start := time.Date(2026, 10, 4, 9, 0, 0, 500, time.FixedZone("KST", 9*3600))
	input := "2026-10-03 10:23:02, Error old failure\n2026-10-04 09:00:00, Info beginning\ncontinuation\n2026-10-04 09:00:03, Error missing payload\n2026-10-05 18:25:33, Error unrelated\nextra\n"
	got, err := extractDiagnosticWindow(strings.NewReader(input), start, start.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"beginning", "continuation", "missing payload"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s: %s", want, got)
		}
	}
	for _, unwanted := range []string{"old failure", "unrelated", "extra"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("included %s", unwanted)
		}
	}
}

func TestDiagnosticsIncludeRotatedLogsAndMissingSources(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Logs", "CBS")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 4, 9, 0, 0, 0, time.Local)
	p := filepath.Join(dir, "CbsPersist_20261004100000.log")
	if err := os.WriteFile(p, []byte("2026-10-04 09:00:02, Info needed component\n2026-10-05 09:00:02, Error wrong date\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, start.Add(time.Hour), start.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	report := string(collectDiagnostics(root, start, start.Add(time.Minute), CommandSpec{Name: "DISM.exe", Args: []string{"/RestoreHealth"}}, errors.New("0x800f0915"), "console evidence"))
	for _, want := range []string{"needed component", "Read unavailable", "0x800f0915", "console evidence", "DISM.exe /RestoreHealth"} {
		if !strings.Contains(report, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(report, "wrong date") {
		t.Fatal("unrelated date included")
	}
}

func TestDiagnosticTailRetainsFailureWithExplicitTruncation(t *testing.T) {
	var b diagnosticTail
	b.Write([]byte(strings.Repeat("x", diagnosticLimit+100)))
	b.Write([]byte("FINAL FAILURE"))
	if len(b.data) > diagnosticLimit || !strings.HasSuffix(b.String(), "FINAL FAILURE") || !strings.Contains(b.String(), "TRUNCATED") {
		t.Fatal("tail was not retained or marked")
	}
}

func TestFailureEnablesDebugAndSuccessDoesNot(t *testing.T) {
	t.Setenv("SystemRoot", "")
	for _, fail := range []int{0, 1, 2} {
		app := &App{ctx: context.Background(), runner: &recordingRunner{errAt: fail}}
		var last RepairEvent
		app.emitEvent = func(e RepairEvent) { last = e }
		app.runRepair(context.Background())
		if last.DebugAvailable != (fail > 0) || (len(app.diagnosticReport) > 0) != (fail > 0) {
			t.Fatalf("failure=%d, event=%+v", fail, last)
		}
		if fail > 0 && !strings.Contains(string(app.diagnosticReport), "50.0%") {
			t.Fatal("command output missing")
		}
	}
}
