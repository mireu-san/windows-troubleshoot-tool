//go:build !windows

package main

func openWindowsUpdateTroubleshooter() error { return errUnsupportedPlatform }
func openWindowsUpdateSettings() error       { return errUnsupportedPlatform }
