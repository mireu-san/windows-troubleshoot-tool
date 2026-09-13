package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAppUpdate(t *testing.T) {
	for _, tc := range []struct {
		name, current, body          string
		status                       int
		available, download, wantErr bool
	}{
		{"revision upgrade", "2026.09.12", releaseJSON("v2026.09.12.1", "WindowsSystemRepairHelper.exe", "github.com"), 200, true, true, false},
		{"revision numeric comparison", "2026.09.12.2", releaseJSON("v2026.09.12.10", "WindowsSystemRepairHelper.exe", "github.com"), 200, true, true, false},
		{"revision same", "2026.09.12.1", releaseJSON("v2026.09.12.1", "WindowsSystemRepairHelper.exe", "github.com"), 200, false, false, false},
		{"revision no downgrade", "2026.09.12.1", releaseJSON("v2026.09.12", "WindowsSystemRepairHelper.exe", "github.com"), 200, false, false, false},
		{"implicit zero revision", "2026.09.12", releaseJSON("v2026.09.12.0", "WindowsSystemRepairHelper.exe", "github.com"), 200, false, false, false},
		{"date version", "2026.09.12", releaseJSON("v2026.10.01", "WindowsSystemRepairHelper.exe", "github.com"), 200, true, true, false},
		{"same date", "2026.09.12", releaseJSON("v2026.09.12", "WindowsSystemRepairHelper.exe", "github.com"), 200, false, false, false},
		{"numeric comparison", "1.9.0", releaseJSON("v1.10.0", "WindowsSystemRepairHelper.exe", "github.com"), 200, true, true, false},
		{"same version", "1.10.0", releaseJSON("v1.10.0", "WindowsSystemRepairHelper.exe", "github.com"), 200, false, false, false},
		{"no downgrade", "2.0.0", releaseJSON("v1.10.0", "WindowsSystemRepairHelper.exe", "github.com"), 200, false, false, false},
		{"installer", "1.0.0", releaseJSON("v1.1.0", "WindowsSystemRepairHelper-amd64-installer.exe", "github.com"), 200, true, true, false},
		{"wrong asset", "1.0.0", releaseJSON("v1.1.0", "other.exe", "github.com"), 200, true, false, false},
		{"untrusted URL", "1.0.0", releaseJSON("v1.1.0", "WindowsSystemRepairHelper.exe", "example.com"), 200, true, false, false},
		{"no release", "1.0.0", `{}`, 404, false, false, false},
		{"forbidden", "1.0.0", `{}`, 403, false, false, true},
		{"rate limited", "1.0.0", `{}`, 429, false, false, true},
		{"server error", "1.0.0", `{}`, 500, false, false, true},
		{"invalid JSON", "1.0.0", `{`, 200, false, false, true},
		{"invalid tag", "1.0.0", `{"tag_name":"latest"}`, 200, false, false, true},
		{"prerelease", "1.0.0", `{"tag_name":"v2.0.0","prerelease":true}`, 200, false, false, true},
		{"draft", "1.0.0", `{"tag_name":"v2.0.0","draft":true}`, 200, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Accept") != "application/vnd.github+json" {
					t.Error("missing API accept header")
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			info, err := checkAppUpdate(context.Background(), server.Client(), server.URL, tc.current)
			if (err != nil) != tc.wantErr || info.Available != tc.available || info.CanDownload != tc.download {
				t.Fatalf("info=%+v, err=%v", info, err)
			}
			if err == nil && info.Message == "" {
				t.Error("missing user message")
			}
		})
	}
}

func releaseJSON(tag, name, host string) string {
	return fmt.Sprintf(`{"tag_name":%q,"assets":[{"name":%q,"state":"uploaded","browser_download_url":%q}]}`, tag, name,
		"https://"+host+"/mireu-san/windows-troubleshoot-tool/releases/download/"+tag+"/"+name)
}

func TestParseAppVersionRejectsInvalid(t *testing.T) {
	for _, version := range []string{"", "latest", "1.2", "1.2.3-beta", "1.2.3.4.5", "1.2.3.", "1.2.3.-1", "-1.2.3", "999999999999999999999.0.0"} {
		if _, err := parseAppVersion(version); err == nil {
			t.Errorf("accepted %q", version)
		}
	}
}

func TestAppUpdateCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := checkAppUpdate(ctx, http.DefaultClient, latestReleaseAPI, "1.0.0"); err == nil {
		t.Fatal("expected cancellation error")
	}
}

func TestDownloadRequiresCheck(t *testing.T) {
	if err := NewApp().DownloadAppUpdate(); err == nil {
		t.Fatal("download must require a checked release")
	}
}

func TestDownloadURLValidation(t *testing.T) {
	base := releasesURL + "/download/v1.1.0/WindowsSystemRepairHelper.exe"
	for _, raw := range []string{
		"http://github.com/mireu-san/windows-troubleshoot-tool/releases/download/v1.1.0/WindowsSystemRepairHelper.exe",
		base + "?redirect=evil", base + "#fragment",
		"https://github.com/other/repo/releases/download/v1.1.0/WindowsSystemRepairHelper.exe",
		"https://user@github.com/mireu-san/windows-troubleshoot-tool/releases/download/v1.1.0/WindowsSystemRepairHelper.exe",
	} {
		if validAppDownloadURL(raw, "v1.1.0", "WindowsSystemRepairHelper.exe") {
			t.Errorf("accepted %q", raw)
		}
	}
}

func TestDownloadUpdateVerification(t *testing.T) {
	payload := []byte("test release payload")
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(payload))
	for _, tc := range []struct {
		name, digest string
		status       int
		wantErr      bool
	}{
		{"valid", digest, 200, false},
		{"missing digest", "", 200, true},
		{"mismatch", fmt.Sprintf("sha256:%064d", 0), 200, true},
		{"HTTP error", digest, 404, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); w.Write(payload) }))
			defer server.Close()
			path, err := downloadUpdate(context.Background(), server.Client(), server.URL, tc.digest)
			if (err != nil) != tc.wantErr {
				t.Fatalf("path=%s err=%v", path, err)
			}
			if path != "" {
				defer os.RemoveAll(filepath.Dir(path))
				got, err := os.ReadFile(path)
				if err != nil || string(got) != string(payload) {
					t.Fatal("download contents differ")
				}
			}
		})
	}
}

func TestInstallUpdateRequiresReadyAndIdle(t *testing.T) {
	for _, app := range []*App{{}, {running: true, stagedUpdate: "test.exe"}, {downloadingUpdate: true, stagedUpdate: "test.exe"}} {
		if err := app.InstallAppUpdate(); err == nil {
			t.Fatal("installation should be rejected")
		}
		if app.installingUpdate {
			t.Fatal("failed installation locked the app")
		}
	}
}
