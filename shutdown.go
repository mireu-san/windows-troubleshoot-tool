package main

import "context"

// Called only after both repair commands succeed. Serialize scheduling and cancellation.
func (a *App) scheduleRepairShutdown(ctx context.Context) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.shutdownAfterRepair || ctx.Err() != nil {
		return false, nil
	}
	if err := a.executeShutdownCommand(false); err != nil {
		return false, err
	}
	a.shutdownPending = true
	return true, nil
}

func (a *App) CancelScheduledShutdown() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// Allow cancellation after reopening the app, too: Windows owns the timer.
	if err := a.executeShutdownCommand(true); err != nil {
		return err
	}
	a.shutdownPending = false
	a.shutdownAfterRepair = false
	return nil
}

func (a *App) executeShutdownCommand(abort bool) error {
	if a.shutdownCommand != nil {
		return a.shutdownCommand(abort)
	}
	return runShutdownCommand(abort)
}
