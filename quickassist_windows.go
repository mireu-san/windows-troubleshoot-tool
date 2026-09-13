//go:build windows

package main

func openQuickAssist() error {
	// Quick Assist registers this URI when the Microsoft Store app is installed.
	return openWindowsURI("ms-quick-assist:")
}
