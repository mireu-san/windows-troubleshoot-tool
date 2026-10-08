//go:build windows

package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed graphics/scan.ps1
var graphicsScanScript string

func graphicsPowerShell(ctx context.Context, script string) ([]byte, error) {
	units := utf16.Encode([]rune(script))
	data := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(data[i*2:], u)
	}
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, path, "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(data))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("그래픽 정보 수집 시간 초과 또는 취소: %w", ctx.Err())
	}
	if err != nil {
		return nil, fmt.Errorf("Windows 그래픽 정보 조회 실패: %w", err)
	}
	return []byte(strings.TrimPrefix(string(out), "\ufeff")), nil
}
func scanGraphicsSystem(ctx context.Context, start, end time.Time) (*GraphicsReport, error) {
	script := strings.ReplaceAll(strings.ReplaceAll(graphicsScanScript, "__START__", start.Format(time.RFC3339)), "__END__", end.Format(time.RFC3339))
	data, err := graphicsPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	var report GraphicsReport
	if err = json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("그래픽 정보 해석 실패: %w", err)
	}
	report.Monitors, err = graphicsMonitors()
	if err != nil {
		report.Issues = append(report.Issues, err.Error())
	}
	if len(report.GPUs) == 0 {
		report.Issues = append(report.Issues, "No display adapters returned; detection is incomplete")
	}
	return &report, nil
}

func updaterCandidates(vendor string) []string {
	roots := []string{os.Getenv("ProgramW6432"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")}
	var relative string
	switch vendor {
	case "intel":
		relative = filepath.Join("Intel", "Driver and Support Assistant", "DSATray.exe")
	case "amd":
		relative = filepath.Join("AMD", "CNext", "CNext", "RadeonSoftware.exe")
	case "nvidia":
		relative = filepath.Join("NVIDIA Corporation", "NVIDIA app", "CEF", "NVIDIA app.exe")
	default:
		return nil
	}
	var paths []string
	for _, root := range roots {
		if root != "" {
			paths = append(paths, filepath.Join(root, relative))
		}
	}
	return paths
}
func launchGraphicsUpdater(ctx context.Context, vendor string) (bool, error) {
	// Intel's documented scan interface is the browser-based Support Assistant.
	if vendor == "intel" {
		return false, nil
	}
	publisher := map[string]string{"amd": "'Advanced Micro Devices, Inc.','Advanced Micro Devices, Inc','Advanced Micro Devices Inc.','Advanced Micro Devices Inc'", "nvidia": "'NVIDIA Corporation'"}[vendor]
	if publisher == "" {
		return false, fmt.Errorf("지원하지 않는 그래픽 제조사입니다")
	}
	for _, path := range updaterCandidates(vendor) {
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, err
		}
		if info.IsDir() {
			continue
		}
		escaped := strings.ReplaceAll(path, "'", "''")
		script := fmt.Sprintf("$ErrorActionPreference='Stop'; $s=Get-AuthenticodeSignature -LiteralPath '%s'; if ($s.Status -ne 'Valid') { exit 1 }; $name=$s.SignerCertificate.GetNameInfo([System.Security.Cryptography.X509Certificates.X509NameType]::SimpleName,$false); if (@(%s) -notcontains $name) { exit 1 }; Write-Output 'VALID'", escaped, publisher)
		out, err := graphicsPowerShell(ctx, script)
		if err != nil || strings.TrimSpace(string(out)) != "VALID" {
			return false, fmt.Errorf("제조사 업데이트 도구의 전자서명을 확인하지 못했습니다: %s", path)
		}
		cmd := exec.Command(path)
		if err = cmd.Start(); err != nil {
			return false, err
		}
		go func() { _ = cmd.Wait() }()
		return true, nil
	}
	return false, nil
}
func openGraphicsSettings() error { return openWindowsURI("ms-settings:display-advanced") }
func openDeviceManager() error {
	cmd := exec.Command(filepath.Join(os.Getenv("SystemRoot"), "System32", "mmc.exe"), filepath.Join(os.Getenv("SystemRoot"), "System32", "devmgmt.msc"))
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// EnumDisplayDevices supplies the adapter's PnP ID and each attached monitor.
// Current display settings belong to this display path, not to the CPU model.
type graphicsDisplayDevice struct {
	Size        uint32
	Name        [32]uint16
	Description [128]uint16
	Flags       uint32
	ID          [128]uint16
	Key         [128]uint16
}
type graphicsDevMode struct {
	DeviceName                                                                                     [32]uint16
	SpecVersion, DriverVersion, Size, DriverExtra                                                  uint16
	Fields                                                                                         uint32
	PositionX, PositionY                                                                           int32
	Orientation, FixedOutput                                                                       uint32
	Color, Duplex, YResolution, TTOption, Collate                                                  int16
	FormName                                                                                       [32]uint16
	LogPixels                                                                                      uint16
	BitsPerPel, Width, Height, DisplayFlags, Frequency                                             uint32
	ICMMethod, ICMIntent, MediaType, DitherType, Reserved1, Reserved2, PanningWidth, PanningHeight uint32
}

func graphicsMonitors() ([]GraphicsMonitor, error) {
	dll := windows.NewLazySystemDLL("user32.dll")
	enumerate := dll.NewProc("EnumDisplayDevicesW")
	settings := dll.NewProc("EnumDisplaySettingsW")
	if err := enumerate.Find(); err != nil {
		return nil, err
	}
	if err := settings.Find(); err != nil {
		return nil, err
	}
	monitors := []GraphicsMonitor{}
	for i := uintptr(0); i < 64; i++ {
		var adapter graphicsDisplayDevice
		adapter.Size = uint32(unsafe.Sizeof(adapter))
		ok, _, _ := enumerate.Call(0, i, uintptr(unsafe.Pointer(&adapter)), 0)
		if ok == 0 {
			break
		}
		if adapter.Flags&1 == 0 {
			continue
		}
		name := windows.UTF16ToString(adapter.Name[:])
		namePtr, _ := windows.UTF16PtrFromString(name)
		var mode graphicsDevMode
		mode.Size = uint16(unsafe.Sizeof(mode))
		modeOK, _, _ := settings.Call(uintptr(unsafe.Pointer(namePtr)), uintptr(0xffffffff), uintptr(unsafe.Pointer(&mode)))
		for j := uintptr(0); j < 32; j++ {
			var monitor graphicsDisplayDevice
			monitor.Size = uint32(unsafe.Sizeof(monitor))
			ok, _, _ := enumerate.Call(uintptr(unsafe.Pointer(namePtr)), j, uintptr(unsafe.Pointer(&monitor)), 0)
			if ok == 0 {
				break
			}
			if monitor.Flags&1 == 0 {
				continue
			}
			m := GraphicsMonitor{ID: windows.UTF16ToString(monitor.ID[:]), Name: windows.UTF16ToString(monitor.Description[:]), Adapter: windows.UTF16ToString(adapter.Description[:]), AdapterID: windows.UTF16ToString(adapter.ID[:]), Device: name}
			if modeOK != 0 {
				m.X, m.Y, m.Primary, m.ModeKnown = mode.PositionX, mode.PositionY, adapter.Flags&4 != 0, true
				m.Width = mode.Width
				m.Height = mode.Height
				if mode.Frequency > 1 {
					m.Hz = mode.Frequency
				}
			}
			monitors = append(monitors, m)
		}
	}
	if len(monitors) == 0 {
		return monitors, fmt.Errorf("Monitor paths unavailable; remote sessions or disabled displays may not expose them")
	}
	return monitors, nil
}

//go:embed graphics/comprehensive.ps1
var flickerDetailsScript string

func collectFlickerDetails(ctx context.Context, start, end time.Time) (*FlickerDetails, error) {
	script := strings.ReplaceAll(strings.ReplaceAll(flickerDetailsScript, "__START__", start.Format(time.RFC3339)), "__END__", end.Format(time.RFC3339))
	data, err := graphicsPowerShell(ctx, script)
	if err != nil {
		return nil, err
	}
	var details FlickerDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}
	return &details, nil
}
