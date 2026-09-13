//go:build !windows

package main

import (
	"context"
	"errors"
)

var errUnsupportedPlatform = errors.New("unsupported platform")

type unsupportedRunner struct{}

func newSystemRunner() commandRunner { return &unsupportedRunner{} }

func (r *unsupportedRunner) Run(_ context.Context, _ CommandSpec, _ func(string, int)) error {
	return errUnsupportedPlatform
}
