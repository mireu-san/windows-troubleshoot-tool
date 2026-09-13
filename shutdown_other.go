//go:build !windows

package main

func runShutdownCommand(bool) error { return errUnsupportedPlatform }
