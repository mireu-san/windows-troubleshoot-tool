//go:build windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func verifyUpdateFile(path, digest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	if "sha256:"+hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(digest) {
		return errors.New("업데이트 파일 검증에 실패했습니다")
	}
	return nil
}

func launchAppUpdate(path, digest, name string) error {
	if err := verifyUpdateFile(path, digest); err != nil {
		return err
	}
	var cmd *exec.Cmd
	if name == "WindowsSystemRepairHelper-amd64-installer.exe" {
		cmd = exec.Command(path)
	} else {
		current, err := os.Executable()
		if err != nil {
			return err
		}
		// Run the current, trusted updater code from a separate executable so the
		// installed executable can be replaced after this process has exited.
		helper := filepath.Join(filepath.Dir(path), "updater.exe")
		if err := copyUpdateFile(current, helper); err != nil {
			return err
		}
		cmd = exec.Command(helper, "--apply-app-update", current, path, strconv.Itoa(os.Getpid()), digest)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func copyUpdateFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(target)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(target)
		return closeErr
	}
	return nil
}

func handleAppUpdate() bool {
	if len(os.Args) < 2 || os.Args[1] != "--apply-app-update" {
		return false
	}
	if len(os.Args) != 6 {
		return true
	}
	target, staged, pidText, digest := os.Args[2], os.Args[3], os.Args[4], os.Args[5]
	if err := applyAppUpdate(target, staged, pidText, digest); err != nil {
		// The original EXE remains available (or was restored) on failure.
		os.WriteFile(filepath.Join(filepath.Dir(staged), "update-error.txt"), []byte(err.Error()), 0600)
		title, _ := windows.UTF16PtrFromString("앱 업데이트 실패")
		message, _ := windows.UTF16PtrFromString("업데이트를 적용하지 못했습니다. 기존 앱을 다시 실행해 주세요.\n\n" + err.Error())
		windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)), 0x10)
	}
	return true
}

func applyAppUpdate(target, staged, pidText, digest string) error {
	pid, err := strconv.ParseUint(pidText, 10, 32)
	if err != nil {
		return err
	}
	process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err == nil {
		result, waitErr := windows.WaitForSingleObject(process, 60000)
		windows.CloseHandle(process)
		if waitErr != nil {
			return waitErr
		}
		if result != windows.WAIT_OBJECT_0 {
			return errors.New("앱 종료를 기다리다 시간이 초과되었습니다")
		}
	} else if !errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return err
	}
	if err := verifyUpdateFile(staged, digest); err != nil {
		return err
	}
	// Keep the candidate on the same volume so the final rename is atomic.
	temp, err := os.CreateTemp(filepath.Dir(target), ".app-update-*.exe")
	if err != nil {
		return err
	}
	candidate := temp.Name()
	temp.Close()
	os.Remove(candidate)
	defer os.Remove(candidate)
	if err := copyUpdateFile(staged, candidate); err != nil {
		return err
	}
	if err := verifyUpdateFile(candidate, digest); err != nil {
		return err
	}
	backup := candidate + ".previous"
	if err := os.Rename(target, backup); err != nil {
		return err
	}
	if err := os.Rename(candidate, target); err != nil {
		if restoreErr := os.Rename(backup, target); restoreErr != nil {
			return errors.New("이전 파일 복원 실패: " + backup)
		}
		return err
	}
	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	if err := cmd.Start(); err != nil {
		os.Remove(target)
		if restoreErr := os.Rename(backup, target); restoreErr != nil {
			return errors.New("이전 파일 복원 실패: " + backup)
		}
		return err
	}
	cmd.Process.Release()
	os.Remove(backup)
	os.Remove(staged)
	return nil
}
