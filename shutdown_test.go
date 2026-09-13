package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestShutdownOnlyAfterSuccessfulRepair(t *testing.T) {
	for _, tc := range []struct {
		name     string
		selected bool
		failure  int
		want     bool
	}{
		{"selected success", true, 0, true},
		{"not selected", false, 0, false},
		{"DISM failure", true, 1, false},
		{"SFC failure", true, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &recordingRunner{errAt: tc.failure}
			calls := 0
			var last RepairEvent
			a := &App{runner: runner, shutdownAfterRepair: tc.selected,
				emitEvent: func(e RepairEvent) { last = e },
				shutdownCommand: func(abort bool) error {
					calls++
					if abort || len(runner.commands) != 2 {
						t.Fatal("shutdown before both commands completed")
					}
					return nil
				},
			}
			a.runRepair(context.Background())
			if (calls == 1) != tc.want || a.shutdownPending != tc.want || last.ShutdownScheduled != tc.want {
				t.Fatalf("calls=%d pending=%v event=%+v", calls, a.shutdownPending, last)
			}
			if a.shutdownAfterRepair {
				t.Fatal("option retained after repair")
			}
		})
	}
}

func TestCancelledRepairDoesNotScheduleShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &App{runner: &cancellingRunner{cancel: cancel}, shutdownAfterRepair: true,
		shutdownCommand: func(bool) error { t.Fatal("scheduled after cancellation"); return nil },
	}
	a.runRepair(ctx)
	if a.shutdownPending {
		t.Fatal("shutdown pending")
	}
}

func TestShutdownSchedulingFailurePreservesRepairSuccess(t *testing.T) {
	var last RepairEvent
	a := &App{runner: &recordingRunner{}, shutdownAfterRepair: true,
		emitEvent:       func(e RepairEvent) { last = e },
		shutdownCommand: func(bool) error { return errors.New("test scheduling failure") },
	}
	a.runRepair(context.Background())
	if last.Stage != "complete" || last.ShutdownScheduled || a.shutdownPending || !strings.Contains(last.Message, "자동 종료를 예약하지 못했습니다") {
		t.Fatalf("unexpected result: %+v", last)
	}
}

func TestPendingShutdownBlocksOperationsAndCanBeCancelled(t *testing.T) {
	a := &App{shutdownPending: true, shutdownCommand: func(abort bool) error {
		if !abort {
			t.Fatal("expected abort")
		}
		return nil
	}}
	if a.beginOperation() {
		t.Fatal("operation allowed during shutdown")
	}
	if a.StartRepair() == nil {
		t.Fatal("repair allowed during shutdown")
	}
	if a.InstallAppUpdate() == nil {
		t.Fatal("update allowed during shutdown")
	}
	if err := a.CancelScheduledShutdown(); err != nil {
		t.Fatal(err)
	}
	if a.shutdownPending || !a.beginOperation() {
		t.Fatal("operation blocked after cancellation")
	}
}

func TestShutdownCancellationFailureRetainsPendingState(t *testing.T) {
	a := &App{shutdownPending: true, shutdownCommand: func(bool) error { return errors.New("abort failed") }}
	if a.CancelScheduledShutdown() == nil || !a.shutdownPending {
		t.Fatal("lost pending shutdown after failed abort")
	}
}

func TestCancelShutdownAlsoDisarmsRepairOption(t *testing.T) {
	a := &App{shutdownAfterRepair: true, shutdownCommand: func(bool) error { return nil }}
	if err := a.CancelScheduledShutdown(); err != nil {
		t.Fatal(err)
	}
	if a.shutdownAfterRepair {
		t.Fatal("repair still armed for shutdown")
	}
}
