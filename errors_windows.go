//go:build windows

package main

import "errors"

var errUnsupportedPlatform = errors.New("unsupported platform")
