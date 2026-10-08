//go:build !windows

package main

import (
	"context"
	"time"
)

func collectHardware(context.Context, time.Time, time.Time, HardwareRequest) (*HardwareReport, error) {
	return nil, errUnsupportedPlatform
}
func hardwareDesktop(context.Context) (string, error) { return "", errUnsupportedPlatform }
func openHardwareFolder(string) error                 { return errUnsupportedPlatform }
