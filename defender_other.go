//go:build !windows

package main

import "context"

func runDefenderQuickScan(context.Context) error { return errUnsupportedPlatform }
func openDefenderSecurity() error                { return errUnsupportedPlatform }
