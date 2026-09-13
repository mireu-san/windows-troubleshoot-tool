//go:build windows

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var percentPattern = regexp.MustCompile(`(?i)(\d{1,3}(?:[\.,]\d+)?)\s*%`)

type systemRunner struct{}

func newSystemRunner() commandRunner { return &systemRunner{} }

func (r *systemRunner) Run(ctx context.Context, spec CommandSpec, onOutput func(string, int)) error {
	// chcp 65001 makes localized command output safe to send to the UTF-8 web UI.
	commandLine := quoteWindowsCommand(spec.Name, spec.Args)
	cmd := exec.CommandContext(ctx, "cmd.exe", "/D", "/S", "/C", "chcp 65001>nul & "+commandLine)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	// cmd.exe owns the DISM/SFC child; cancel the process tree, not just the shell.
	cmd.Cancel = func() error {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stop := exec.CommandContext(stopCtx, "taskkill.exe", "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F")
		stop.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return stop.Run()
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	lines := make(chan string, 64)
	read := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		scanner.Split(splitConsoleOutput)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			if line := strings.TrimSpace(scanner.Text()); line != "" {
				lines <- line
			}
		}
		lines <- ""
	}
	go read(stdout)
	go read(stderr)

	finishedReaders := 0
	for finishedReaders < 2 {
		line := <-lines
		if line == "" {
			finishedReaders++
			continue
		}
		onOutput(line, mapProgress(line, spec.ProgressMin, spec.ProgressMax))
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 3010 {
			onOutput("복구를 적용하려면 Windows를 다시 시작해야 합니다.", spec.ProgressMax)
			return nil
		}
		return fmt.Errorf("%s 종료 오류: %w", spec.Name, err)
	}
	return nil
}

func quoteWindowsCommand(name string, args []string) string {
	// All values originate from fixed CommandSpec literals, never from user input.
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func mapProgress(line string, min, max int) int {
	match := percentPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return min
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
	if err != nil {
		return min
	}
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return min + int((float64(max-min)*value)/100)
}

func splitConsoleOutput(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == '\r' || b == '\n' {
			advance = i + 1
			if b == '\r' && advance < len(data) && data[advance] == '\n' {
				advance++
			}
			return advance, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}
