//go:build !windows

package main

import "context"

func findSystemRepairSource(context.Context, string) (string, error) {
	return "", errUnsupportedPlatform
}
func openWindowsRepairSettings() error { return errUnsupportedPlatform }
