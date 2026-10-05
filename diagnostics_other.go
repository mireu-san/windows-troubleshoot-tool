//go:build !windows

package main

import "errors"

func expandDiagnosticArchive(string) ([]string, func(), error) {
	return nil, func() {}, errors.New("CBS CAB extraction requires Windows")
}
