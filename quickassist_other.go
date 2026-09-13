//go:build !windows

package main

func openQuickAssist() error {
	return errUnsupportedPlatform
}
