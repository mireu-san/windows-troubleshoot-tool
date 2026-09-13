package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type recordingRunner struct {
	commands []CommandSpec
	errAt    int
}

func (r *recordingRunner) Run(_ context.Context, spec CommandSpec, output func(string, int)) error {
	r.commands = append(r.commands, spec)
	output("50.0%", (spec.ProgressMin+spec.ProgressMax)/2)
	if r.errAt == len(r.commands) {
		return errors.New("test error")
	}
	return nil
}

func TestRepairCommandOrder(t *testing.T) {
	runner := &recordingRunner{}
	app := &App{ctx: context.Background(), runner: runner}
	app.runRepair(context.Background())

	got := []string{runner.commands[0].Name, runner.commands[1].Name}
	want := []string{"DISM.exe", "sfc.exe"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command order = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(runner.commands[0].Args, []string{"/Online", "/Cleanup-Image", "/RestoreHealth"}) {
		t.Fatalf("unexpected DISM args: %v", runner.commands[0].Args)
	}
	if !reflect.DeepEqual(runner.commands[1].Args, []string{"/scannow"}) {
		t.Fatalf("unexpected SFC args: %v", runner.commands[1].Args)
	}
}

func TestStopsAfterFailure(t *testing.T) {
	runner := &recordingRunner{errAt: 1}
	app := &App{ctx: context.Background(), runner: runner}
	app.runRepair(context.Background())
	if len(runner.commands) != 1 {
		t.Fatalf("ran %d commands, want 1", len(runner.commands))
	}
}

func TestKoreanDuration(t *testing.T) {
	if got := koreanDuration(2*time.Minute + 7*time.Second); got != "2분 7초" {
		t.Fatalf("duration = %q", got)
	}
}

type cancellingRunner struct {
	calls  int
	cancel context.CancelFunc
}

func (r *cancellingRunner) Run(ctx context.Context, _ CommandSpec, _ func(string, int)) error {
	r.calls++
	r.cancel()
	return ctx.Err()
}

func TestRepairCancellationStopsBeforeSFC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := &cancellingRunner{cancel: cancel}
	var final RepairEvent
	app := &App{ctx: ctx, runner: runner, running: true, repairCancel: cancel, emitEvent: func(e RepairEvent) { final = e }}
	app.runRepair(ctx)
	if runner.calls != 1 || final.Stage != "cancelled" {
		t.Fatalf("calls=%d, final=%+v", runner.calls, final)
	}
	if app.IsRunning() || app.repairCancel != nil {
		t.Fatal("operation was not released")
	}
}

func TestCancelRepairDoesNotCancelOtherOperation(t *testing.T) {
	app := &App{running: true}
	app.CancelRepair()
	if !app.IsRunning() {
		t.Fatal("unrelated operation was cleared")
	}
	if err := app.StartDefenderQuickScan(); err == nil {
		t.Fatal("concurrent scan accepted")
	}
}
