package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type repairExit uint32

func (e repairExit) Error() string { return fmt.Sprintf("exit status 0x%x", uint32(e)) }
func (e repairExit) ExitCode() int { return int(int32(e)) }

func TestRepairSourceErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		name string
		code uint32
		want bool
	}{
		{"DISM.exe", 0x800f0915, true}, {"dism.exe", 0x800f081f, true},
		{"sfc.exe", 0x800f0915, false}, {"DISM.exe", 5, false}, {"DISM.exe", 3010, false},
	} {
		err := fmt.Errorf("wrapped: %w", repairExit(tc.code))
		if got := isRepairSourceMissing(CommandSpec{Name: tc.name}, err); got != tc.want {
			t.Fatalf("%+v: %v", tc, got)
		}
	}
	if isRepairSourceMissing(CommandSpec{Name: "DISM.exe"}, errors.New("output mentions 0x800f0915")) {
		t.Fatal("classified untrusted output")
	}
}

func TestSelectCompatibleRepairSource(t *testing.T) {
	current := repairImage{Edition: "Professional", Architecture: 9, Major: 10, Build: 26300, Revision: 9550, Languages: []string{"ko-KR"}}
	good := current
	good.Path = `D:\media & files\install.wim`
	good.Index = 6
	for _, tc := range []struct {
		name   string
		change func(*repairImage)
		want   bool
	}{
		{"matching", func(*repairImage) {}, true},
		{"newer revision", func(i *repairImage) { i.Revision++ }, true},
		{"old revision", func(i *repairImage) { i.Revision-- }, false},
		{"retail build for insider", func(i *repairImage) { i.Build = 26100 }, false},
		{"edition", func(i *repairImage) { i.Edition = "Core" }, false},
		{"architecture", func(i *repairImage) { i.Architecture = 12 }, false},
		{"language", func(i *repairImage) { i.Languages = []string{"en-US"} }, false},
		{"zero index", func(i *repairImage) { i.Index = 0 }, false},
		{"invalid extension", func(i *repairImage) { i.Path = "D:\\setup.exe" }, false},
		{"esd", func(i *repairImage) { i.Path = "D:\\install.esd" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := good
			tc.change(&candidate)
			got, err := selectRepairSource(repairSourceInventory{Current: current, Images: []repairImage{candidate}})
			if err != nil || (got != "") != tc.want {
				t.Fatalf("source=%q err=%v", got, err)
			}
			if tc.want && !strings.HasSuffix(got, ":6") {
				t.Fatal(got)
			}
		})
	}
	current.Languages = append(current.Languages, "en-US")
	got, _ := selectRepairSource(repairSourceInventory{Current: current, Images: []repairImage{good}})
	if got != "" {
		t.Fatal("accepted source missing an installed language")
	}
}

type fallbackRunner struct {
	commands []CommandSpec
	failures map[int]error
}

func (r *fallbackRunner) Run(_ context.Context, spec CommandSpec, output func(string, int)) error {
	r.commands = append(r.commands, spec)
	output("test output", spec.ProgressMin)
	return r.failures[len(r.commands)]
}

func TestSourceFallbackWorkflow(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		retryErr     error
		wantCalls    int
		wantStage    string
		wantSource   bool
	}{
		{"recovered", `wim:D:\sources\install.wim:6`, nil, 3, "complete", false},
		{"no source", "", nil, 1, "error", true},
		{"retry source failure", `wim:D:\sources\install.wim:6`, repairExit(0x800f0915), 2, "error", true},
		{"retry other failure", `wim:D:\sources\install.wim:6`, repairExit(5), 2, "error", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SystemRoot", "")
			runner := &fallbackRunner{failures: map[int]error{1: repairExit(0x800f0915), 2: tc.retryErr}}
			var final RepairEvent
			searches := 0
			app := &App{ctx: context.Background(), runner: runner, running: true, emitEvent: func(e RepairEvent) { final = e }, findRepairSource: func(context.Context, string) (string, error) { searches++; return tc.source, nil }}
			app.runRepair(context.Background())
			if len(runner.commands) != tc.wantCalls || final.Stage != tc.wantStage || final.SourceRequired != tc.wantSource || searches != 1 || app.IsRunning() {
				t.Fatalf("calls=%d searches=%d final=%+v running=%v", len(runner.commands), searches, final, app.IsRunning())
			}
			if tc.wantCalls > 1 {
				args := runner.commands[1].Args
				if !reflect.DeepEqual(args, []string{"/Online", "/Cleanup-Image", "/RestoreHealth", "/Source:" + tc.source, "/English"}) {
					t.Fatalf("args=%v", args)
				}
			}
			if final.Stage == "error" && !strings.Contains(string(app.diagnosticReport), "Initial failure") {
				t.Fatal("initial failure lost")
			}
			if tc.wantStage == "complete" && runner.commands[2].Name != "sfc.exe" {
				t.Fatal("SFC not run after recovery")
			}
		})
	}
}

func TestSourceSearchCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &fallbackRunner{failures: map[int]error{1: repairExit(0x800f0915)}}
	var final RepairEvent
	app := &App{ctx: ctx, runner: runner, running: true, emitEvent: func(e RepairEvent) { final = e }, findRepairSource: func(context.Context, string) (string, error) { cancel(); return "", ctx.Err() }}
	app.runRepair(ctx)
	if len(runner.commands) != 1 || final.Stage != "cancelled" || app.IsRunning() {
		t.Fatalf("final=%+v", final)
	}
}

func TestSelectedSourceDoesNotRepeatOnlineOnlyRepair(t *testing.T) {
	runner := &fallbackRunner{}
	var final RepairEvent
	app := &App{ctx: context.Background(), runner: runner, emitEvent: func(e RepairEvent) { final = e }, findRepairSource: func(_ context.Context, p string) (string, error) {
		if p != "chosen.wim" {
			t.Fatal(p)
		}
		return `wim:C:\chosen.wim:2`, nil
	}}
	app.runRepairWithSource(context.Background(), "chosen.wim")
	if len(runner.commands) != 2 || final.Stage != "complete" || !strings.Contains(strings.Join(runner.commands[0].Args, " "), "/Source:") {
		t.Fatalf("commands=%+v final=%+v", runner.commands, final)
	}
}

func TestIncompatibleSelectedSourceDoesNotRunRepair(t *testing.T) {
	runner := &fallbackRunner{}
	var final RepairEvent
	app := &App{ctx: context.Background(), runner: runner, emitEvent: func(e RepairEvent) { final = e }, findRepairSource: func(context.Context, string) (string, error) { return "", nil }}
	app.runRepairWithSource(context.Background(), "old.wim")
	if len(runner.commands) != 0 || final.Stage != "error" || !final.SourceRequired {
		t.Fatalf("final=%+v", final)
	}
}

func TestSourceFailureExplainsAllRejectedCandidates(t *testing.T) {
	current := repairImage{Edition: "Professional", Architecture: 9, Major: 10, Build: 26300, Revision: 9550, Languages: []string{"ko-KR"}}
	candidate := repairImage{Path: `D:\sources\install.wim`, Index: 6, Edition: "Core", Architecture: 12, Major: 10, Build: 26100, Revision: 1, Languages: []string{"en-US"}}
	inv := repairSourceInventory{Current: current, Images: []repairImage{candidate}, Issues: []string{"E: access denied"}}
	report := repairSourceFailure(inv).Error()
	for _, want := range []string{"26300.9550", "26100.1", "index 6", "edition mismatch", "architecture mismatch", "build mismatch", "revision is older", "missing language: ko-KR", "Inspection failed: E: access denied"} {
		if !strings.Contains(report, want) {
			t.Errorf("missing %q in %s", want, report)
		}
	}
	if !strings.Contains(repairSourceFailure(repairSourceInventory{Current: current}).Error(), "No installation image") {
		t.Fatal("empty inventory should explain absence of media")
	}
}

func TestSourceInspectionFailureReachesUIAndDiagnostics(t *testing.T) {
	t.Setenv("SystemRoot", "")
	runner := &fallbackRunner{failures: map[int]error{1: repairExit(0x800f0915)}}
	var final RepairEvent
	app := &App{ctx: context.Background(), runner: runner, running: true, emitEvent: func(e RepairEvent) { final = e }, findRepairSource: func(context.Context, string) (string, error) {
		return "", errors.New("build mismatch; missing language: ko-KR")
	}}
	app.runRepair(context.Background())
	if final.Stage != "error" || !final.SourceRequired || !strings.Contains(final.SourceDetails, "build mismatch") || len(runner.commands) != 1 {
		t.Fatalf("unexpected fallback: %+v", final)
	}
	if !strings.Contains(string(app.diagnosticReport), final.SourceDetails) {
		t.Fatal("source inspection evidence missing from saved diagnostics")
	}
}
