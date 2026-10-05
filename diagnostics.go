package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const diagnosticLimit = 2 * 1024 * 1024

// Keep the end of each source, where the failure is most likely to appear.
type diagnosticTail struct {
	data      []byte
	truncated bool
}

func (b *diagnosticTail) Write(p []byte) (int, error) {
	n := len(p)
	if len(b.data)+n > diagnosticLimit {
		b.truncated = true
		if n >= diagnosticLimit {
			b.data = append(b.data[:0], p[n-diagnosticLimit:]...)
			return n, nil
		}
		b.data = b.data[len(b.data)+n-diagnosticLimit:]
	}
	b.data = append(b.data, p...)
	return n, nil
}
func (b *diagnosticTail) String() string {
	if b.truncated {
		return "[TRUNCATED: only the last 2 MiB retained; first line may be partial]\n" + string(b.data)
	}
	return string(b.data)
}

// Windows servicing timestamps are local wall-clock time, without a UTC offset.
// Include continuation lines and all severities, not just keyword matches.
func extractDiagnosticWindow(r io.Reader, start, end time.Time) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	var out diagnosticTail
	include := false
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) >= 19 {
			stamp, err := time.ParseInLocation("2006-01-02 15:04:05", line[:19], start.Location())
			if err == nil {
				include = !stamp.Before(start.Truncate(time.Second)) && stamp.Before(end.Truncate(time.Second).Add(time.Second))
			}
		}
		if include {
			fmt.Fprintln(&out, line)
		}
	}
	if len(out.data) == 0 {
		fmt.Fprintln(&out, "[No records in this execution window]")
	}
	return out.String(), scanner.Err()
}

func collectDiagnostics(root string, start, end time.Time, step CommandSpec, failure error, console string) []byte {
	var report bytes.Buffer
	fmt.Fprintf(&report, "Windows Repair Diagnostics\nStarted: %s\nFailed: %s\nTimestamp filter: local Windows time, inclusive seconds\nCommand: %s %s\nFailure: %v\n\n=== Console output ===\n%s\n", start.Format(time.RFC3339), end.Format(time.RFC3339), step.Name, strings.Join(step.Args, " "), failure, console)
	fmt.Fprintln(&report, "System logs may include other Windows servicing work in the same time window. Missing records do not prove absence of corruption. Paths and user names may appear in this report.")
	paths := []string{filepath.Join(root, "Logs", "DISM", "dism.log"), filepath.Join(root, "Logs", "DISM", "dism.log.bak"), filepath.Join(root, "Logs", "CBS", "CBS.log")}
	archives, _ := filepath.Glob(filepath.Join(root, "Logs", "CBS", "CbsPersist*"))
	sort.Slice(archives, func(i, j int) bool { return archives[i] > archives[j] })
	count := 0
	for _, path := range archives {
		info, err := os.Stat(path)
		if err != nil || info.ModTime().Before(start) {
			continue
		}
		if count >= 8 {
			fmt.Fprintln(&report, "[Archive limit: additional CBS archives omitted]")
			break
		}
		paths = append(paths, path)
		count++
	}
	for _, path := range paths {
		fmt.Fprintf(&report, "\n=== %s ===\n", path)
		sources := []string{path}
		cleanup := func() {}
		if strings.EqualFold(filepath.Ext(path), ".cab") {
			var err error
			sources, cleanup, err = expandDiagnosticArchive(path)
			if err != nil {
				fmt.Fprintf(&report, "[Archive unavailable: %v]\n", err)
				continue
			}
		}
		for _, source := range sources {
			f, err := os.Open(source)
			if err != nil {
				fmt.Fprintf(&report, "[Read unavailable: %v]\n", err)
				continue
			}
			content, readErr := extractDiagnosticWindow(f, start, end)
			f.Close()
			fmt.Fprintf(&report, "--- %s ---\n%s", filepath.Base(source), content)
			if readErr != nil {
				fmt.Fprintf(&report, "[Incomplete read: %v]\n", readErr)
			}
		}
		cleanup()
	}
	return report.Bytes()
}

func (a *App) captureRepairFailure(start time.Time, step CommandSpec, failure error, console string) {
	end := time.Now()
	a.emit(RepairEvent{Stage: "debugging", Title: "실패 로그 수집 중", Message: "이번 실행 시간대의 DISM·CBS 로그를 보존하고 있습니다."})
	root := os.Getenv("SystemRoot")
	var report []byte
	if root == "" {
		report = []byte(fmt.Sprintf("Started: %s\nFailed: %s\nCommand: %s %s\nFailure: %v\n%s\n[SystemRoot unavailable: Windows logs were not collected]\n", start.Format(time.RFC3339), end.Format(time.RFC3339), step.Name, strings.Join(step.Args, " "), failure, console))
	} else {
		report = collectDiagnostics(root, start, end, step, failure, console)
	}
	a.mu.Lock()
	a.diagnosticReport = report
	a.mu.Unlock()
}

// Export the captured snapshot, not logs from a later retry or another date.
func (a *App) ExportRepairDiagnostics() (string, error) {
	a.mu.Lock()
	report := a.diagnosticReport
	a.mu.Unlock()
	if len(report) == 0 {
		return "", errors.New("저장할 실패 로그가 없습니다")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: translateNative("실패 로그 저장", a.GetLanguageSettings().Language), DefaultFilename: "repair-diagnostics.txt", Filters: []runtime.FileFilter{{DisplayName: "Text (*.txt)", Pattern: "*.txt"}}})
	if err != nil || path == "" {
		return "", err
	}
	if err = os.WriteFile(path, report, 0600); err != nil {
		return "", err
	}
	return path, nil
}
