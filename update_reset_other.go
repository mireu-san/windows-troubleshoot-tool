//go:build !windows

package main

import "context"

type updateReporter func(stage, title, message, output string, progress int)

func resetWindowsUpdate(_ context.Context, _ updateReporter) error {
	return errUnsupportedPlatform
}
