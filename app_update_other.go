//go:build !windows

package main

func launchAppUpdate(string, string, string) error { return errUnsupportedPlatform }
func handleAppUpdate() bool                        { return false }
