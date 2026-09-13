package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Override at release build time with -ldflags "-X main.appVersion=2026.09.13".
var appVersion = "2026.09.14"

const releasesURL = "https://github.com/mireu-san/windows-troubleshoot-tool/releases"
const latestReleaseAPI = "https://api.github.com/repos/mireu-san/windows-troubleshoot-tool/releases/latest"

type AppUpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Available      bool   `json:"available"`
	CanDownload    bool   `json:"canDownload"`
	Message        string `json:"message"`
	downloadURL    string
	digest         string
	assetName      string
}

type githubRelease struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name   string `json:"name"`
		URL    string `json:"browser_download_url"`
		State  string `json:"state"`
		Digest string `json:"digest"`
	} `json:"assets"`
}

var stableVersion = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)(?:\.([0-9]+))?$`)

func parseAppVersion(version string) ([4]uint64, error) {
	var result [4]uint64
	match := stableVersion.FindStringSubmatch(version)
	if match == nil {
		return result, errors.New("버전은 2026.09.12 또는 2026.09.12.1 형식이어야 합니다")
	}
	for i := range result {
		if match[i+1] == "" {
			continue
		}
		n, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return result, errors.New("버전 번호가 너무 큽니다")
		}
		result[i] = n
	}
	return result, nil
}

func checkAppUpdate(ctx context.Context, client *http.Client, endpoint, current string) (AppUpdateInfo, error) {
	info := AppUpdateInfo{CurrentVersion: current}
	installed, err := parseAppVersion(current)
	if err != nil {
		return info, fmt.Errorf("현재 앱 버전을 확인할 수 없습니다: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "WindowsSystemRepairHelper/"+current)
	resp, err := client.Do(req)
	if err != nil {
		return info, errors.New("업데이트 서버에 연결하지 못했습니다. 인터넷 연결을 확인한 뒤 다시 시도해 주세요")
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotFound:
		info.Message = "공개된 릴리스를 찾을 수 없습니다. 아직 배포되지 않았거나 저장소가 비공개일 수 있습니다."
		return info, nil
	case http.StatusForbidden, http.StatusTooManyRequests:
		return info, errors.New("GitHub 요청이 제한되었습니다. 잠시 후 다시 시도해 주세요")
	case http.StatusOK:
	default:
		return info, fmt.Errorf("업데이트 서버 오류입니다 (HTTP %d). 잠시 후 다시 시도해 주세요", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return info, errors.New("업데이트 정보를 읽을 수 없습니다. 잠시 후 다시 시도해 주세요")
	}
	latest, err := parseAppVersion(release.Tag)
	if err != nil {
		return info, fmt.Errorf("배포 버전을 확인할 수 없습니다: %w", err)
	}
	if release.Draft || release.Prerelease {
		return info, errors.New("정식 릴리스가 아닙니다")
	}
	info.LatestVersion = release.Tag
	for i := range installed {
		if latest[i] != installed[i] {
			info.Available = latest[i] > installed[i]
			break
		}
	}
	if !info.Available {
		info.Message = "현재 최신 버전을 사용하고 있습니다."
		return info, nil
	}
	info.Message = "새 버전 " + release.Tag + "을 사용할 수 있습니다."
	// Exact names avoid accidentally offering another architecture or an unrelated executable.
	for _, name := range []string{"WindowsSystemRepairHelper.exe", "WindowsSystemRepairHelper-amd64-installer.exe"} {
		for _, asset := range release.Assets {
			if asset.Name == name && asset.State == "uploaded" && validAppDownloadURL(asset.URL, release.Tag, name) {
				info.downloadURL = asset.URL
				info.digest = asset.Digest
				info.assetName = asset.Name
				info.CanDownload = true
				return info, nil
			}
		}
	}
	info.Message += " 다운로드 파일이 아직 준비되지 않았습니다. 다음 실행 시 다시 확인합니다."
	return info, nil
}

func validAppDownloadURL(raw, tag, name string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "github.com" && u.User == nil && u.RawQuery == "" && u.Fragment == "" &&
		u.Path == "/mireu-san/windows-troubleshoot-tool/releases/download/"+tag+"/"+name && !strings.Contains(name, "/")
}

func (a *App) GetAppVersion() string { return appVersion }

func (a *App) CheckAppUpdate() (AppUpdateInfo, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	info, err := checkAppUpdate(ctx, &http.Client{Timeout: 15 * time.Second}, latestReleaseAPI, appVersion)
	a.mu.Lock()
	a.appDownloadURL = info.downloadURL
	a.appUpdateDigest = info.digest
	a.appUpdateName = info.assetName
	a.mu.Unlock()
	return info, err
}

// Download to a private staging directory; never launch before explicit approval.
func (a *App) DownloadAppUpdate() error {
	a.mu.Lock()
	if a.downloadingUpdate || a.installingUpdate {
		a.mu.Unlock()
		return errors.New("업데이트 준비 중입니다")
	}
	link, digest := a.appDownloadURL, a.appUpdateDigest
	if link == "" {
		a.mu.Unlock()
		return errors.New("먼저 앱 업데이트를 확인해 주세요")
	}
	a.downloadingUpdate = true
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.downloadingUpdate = false; a.mu.Unlock() }()
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()
	path, err := downloadUpdate(ctx, &http.Client{Timeout: 5 * time.Minute}, link, digest)
	if err != nil {
		return err
	}
	a.mu.Lock()
	old := a.stagedUpdate
	a.stagedUpdate = path
	a.mu.Unlock()
	if old != "" {
		os.RemoveAll(filepath.Dir(old))
	}
	return nil
}

func downloadUpdate(ctx context.Context, client *http.Client, link, digest string) (string, error) {
	expected, err := hex.DecodeString(strings.TrimPrefix(digest, "sha256:"))
	if err != nil || !strings.HasPrefix(digest, "sha256:") || len(expected) != sha256.Size {
		return "", errors.New("배포 파일의 SHA-256 검증 정보가 없습니다. 다음 실행 시 다시 시도합니다")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.New("업데이트 다운로드에 실패했습니다. 인터넷 연결을 확인해 주세요")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("다운로드 오류 (HTTP %d)", resp.StatusCode)
	}
	const maxSize = 256 << 20
	if resp.ContentLength > maxSize {
		return "", errors.New("업데이트 파일이 너무 큽니다")
	}
	dir, err := os.MkdirTemp("", "windows-repair-update-")
	if err != nil {
		return "", err
	}
	success := false
	defer func() {
		if !success {
			os.RemoveAll(dir)
		}
	}()
	path := filepath.Join(dir, "update.exe")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, hash), io.LimitReader(resp.Body, maxSize+1))
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil || n == 0 || n > maxSize {
		return "", errors.New("업데이트 파일을 저장하지 못했습니다")
	}
	if hex.EncodeToString(hash.Sum(nil)) != hex.EncodeToString(expected) {
		return "", errors.New("다운로드 파일 검증에 실패했습니다. 설치하지 않습니다")
	}
	success = true
	return path, nil
}

func (a *App) InstallAppUpdate() error {
	a.mu.Lock()
	if a.running || a.installingUpdate || a.downloadingUpdate || a.shutdownPending {
		a.mu.Unlock()
		return errors.New("검사 또는 복구 작업을 마치고 종료 예약을 취소한 뒤 업데이트를 설치해 주세요")
	}
	path, digest, name := a.stagedUpdate, a.appUpdateDigest, a.appUpdateName
	if path == "" {
		a.mu.Unlock()
		return errors.New("업데이트를 먼저 다운로드해 주세요")
	}
	a.installingUpdate = true
	a.mu.Unlock()
	if err := launchAppUpdate(path, digest, name); err != nil {
		a.mu.Lock()
		a.installingUpdate = false
		a.mu.Unlock()
		return err
	}
	runtime.Quit(a.ctx)
	return nil
}

func (a *App) OpenAppReleases() { runtime.BrowserOpenURL(a.ctx, releasesURL) }
