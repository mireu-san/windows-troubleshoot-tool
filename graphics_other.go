//go:build !windows

package main

import (
	"context"
	"time"
)

func scanGraphicsSystem(context.Context, time.Time, time.Time) (*GraphicsReport, error) {
	return nil, errUnsupportedPlatform
}
func launchGraphicsUpdater(context.Context, string) (bool, error) {
	return false, errUnsupportedPlatform
}
func openGraphicsSettings() error { return errUnsupportedPlatform }
func openDeviceManager() error    { return errUnsupportedPlatform }

func collectFlickerDetails(context.Context, time.Time, time.Time) (*FlickerDetails, error) {
	return nil, errUnsupportedPlatform
}
func captureDisplaySample() (DisplaySample, error) { return DisplaySample{}, errUnsupportedPlatform }
