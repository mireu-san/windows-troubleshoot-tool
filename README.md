# Windows System Repair Helper

A desktop app that lets nontechnical users run Microsoft's recommended Windows repair procedure with one click. The application interface is in Korean; button names below are English translations.

The app runs with administrator privileges and executes these fixed commands in order:

```text
DISM.exe /Online /Cleanup-Image /RestoreHealth
sfc.exe /scannow
```

The app does not accept user-supplied commands. Its repair workflow is designed to repair Windows components and system files without removing personal files or installed applications.

The **Open Quick Assist** button launches Microsoft Quick Assist on Windows 11. If it is not installed, get it from the [Microsoft Store](https://apps.microsoft.com/detail/9p7bp5vnwkx5).

For failed or stalled Windows updates, the app provides a two-step workflow:

1. **Troubleshoot Windows Update** opens Microsoft's automated troubleshooter in the Windows 11 Get Help app.
2. **Reset Update Cache** is intended for problems the troubleshooter cannot resolve. It temporarily stops BITS, Windows Update, and Cryptographic Services, then renames the `SoftwareDistribution` and `catroot2` folders as dated backups. Services are restored to their original state whether the operation succeeds or fails.

## Before You Run

- Save your work and back up important files before starting system maintenance.
- The app runs with administrator privileges and can modify Windows system files, services, and update caches. Use it only on computers you own or are authorized to maintain. For managed work computers, follow your IT support procedures.
- Completing the repair commands does not guarantee that the original problem is resolved. A Windows restart or further troubleshooting may be necessary. Cancelling a repair does not undo changes already applied.
- Automatic shutdown is optional and off by default. If enabled, it can forcibly close applications and lose unsaved work. Closing this app does not cancel a scheduled shutdown; use **Cancel Scheduled Shutdown**.

## Find a Tool for Your Situation

Choose **Find a Tool for My Situation** if you are unsure which button to use. Answer up to three questions about symptoms, internet access and steps already tried. The guide explains why a tool may help, what it does, what to check afterwards and its limits.

Choosing a recommendation takes you to the existing tool button and highlights it. Advanced tools are expanded when necessary. The guide does not start scans or repairs; the normal pre-action explanation and confirmation remain in place. You can go back to change an answer or close the guide at any time.

The guide prioritizes basic Windows Update troubleshooting before cache reset, directs persistent problems to support, and distinguishes a single application's problem from Windows-wide symptoms. If the affected computer cannot reach the desktop, it offers Microsoft's Startup Repair documentation instead of claiming this running app can repair another PC. Answers are held only in memory for this screen; they are not sent to a server or saved to disk. Recommendations are based on answers, not automated diagnosis of the computer.

Frontend decision-tree tests:

```powershell
npm --prefix frontend test
```

## Display Language

The interface supports Korean, English and Japanese. On first launch it follows the current user's Windows display language; other languages fall back to English. This uses the Windows UI language, not the keyboard layout or regional date format. Browser previews use the browser language.

Use **Language** in the header or questionnaire to choose a language, or select **Follow system setting** to return to automatic selection. Changes apply without restarting or interrupting a scan, repair or questionnaire. The desktop app stores only the language preference in `%APPDATA%\WindowsSystemRepairHelper\language.json`; questionnaire answers are not saved. Browser previews store their language preference locally in the browser.

Buttons, questionnaire guidance, confirmations and app-authored status messages are translated. Command output, original license texts and Windows-owned dialogs keep their original language. WebView2 installation guidance and the app's native close confirmation also use the selected language.

## Technology

- Go backend
- Wails v2 desktop shell
- Vite with plain HTML, CSS, and JavaScript
- Windows 10/11 and WebView2 Runtime

Next.js is not used because this single-screen app does not need server rendering or routing, and those features would add build size and complexity.

## Development Setup

Install the following on Windows:

1. Go 1.25 or later
2. Node.js 20 or later
3. WebView2 Runtime, which is usually already installed on Windows 10/11
4. Wails CLI

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
wails doctor
```

## Run and Build

```powershell
wails dev
wails build -clean -webview2 browser
```

The executable is generated in `build/bin/`, using the `outputfilename` setting in `wails.json`. Windows User Account Control (UAC) requests administrator privileges when the app starts.

## Portable Single-File Distribution

Build a Windows executable that runs without a separate application installer:

```powershell
wails build -clean -webview2 browser
```

Distribute the generated executable from `build/bin/`. No application installation or additional files are required. If WebView2 is missing or outdated, the app offers to open Microsoft's official download page. Install WebView2 from Microsoft, then restart this app. The application does not bundle a WebView2 installer or the full runtime. Downloading WebView2 requires an internet connection.

## Publishing App Updates

Whenever the app starts, it checks the latest public stable GitHub Release in `mireu-san/windows-troubleshoot-tool` in the background. The current version appears next to the app title. If a newer version is available, the app automatically downloads the executable, checks it against the SHA-256 digest supplied by the GitHub API, and asks whether to install it.

Choosing **Install** closes the app, replaces the executable, and restarts it. Choosing **Later** keeps the current app running; the update button remains available for installation later. Installation is blocked during scans and repairs. If executable replacement or process startup fails, the previous file is restored. If a release contains only the supported installer, the app launches that installer instead.

The app does not install an update if downloading fails or the verification digest is missing or does not match. Automatic updates require an executable attached to a GitHub Release.

The current default version is `2026.10.05`. Specify the version when building a release on Windows:

```powershell
wails build -clean -platform windows/amd64 -webview2 browser -ldflags "-X main.appVersion=2026.10.05"
```

1. Commit the changes and push them to GitHub.
2. Create a stable release with a tag such as `v2026.10.05`. The tag must match the version embedded in the build. Supported formats are `vYYYY.MM.DD` and `vYYYY.MM.DD.N`; increment the last number for additional releases on the same day. See the compatibility notes below before adopting a four-part tag.
3. Prepare the compatibility executable and ZIP described under **Distribution Filenames**, then attach both to the release. Automatic updates select `WindowsSystemRepairHelper.exe` first. If distributing only an installer, name it `WindowsSystemRepairHelper-amd64-installer.exe`. These naming rules apply to Windows x64 builds.
4. Publish the release as the latest stable release, with the prerelease option disabled.
5. Launch an older app version and verify that it checks for updates, downloads the new executable, and displays the installation prompt.

Pushing source code or creating a tag alone does not distribute an update. A published stable release and its executable attachment are required. The repository must be public for the app to access it without authentication; no GitHub token is embedded in the app. Missing releases or connection failures do not prevent normal app use. The app checks again on its next launch. If an update has no usable download yet, it directs users to the release information.

The API integration follows the [GitHub Releases documentation](https://docs.github.com/en/rest/releases/releases#get-the-latest-release).

## Official References

- [Microsoft: Use the System File Checker tool to repair missing or corrupted system files](https://support.microsoft.com/en-us/windows/experience/backup-recovery/use-the-system-file-checker-tool-to-repair-missing-or-corrupted-system-files)
- [Wails installation guide](https://wails.io/docs/gettingstarted/installation/)

## Defender Quick Scan

The **Defender Quick Scan** button runs Microsoft's [Start-MpScan](https://learn.microsoft.com/en-us/powershell/module/defender/start-mpscan) command with `-ScanType QuickScan`. The app prevents duplicate scans and concurrent repair operations while the scan runs. It does not change Defender settings or exclusions. Use **Windows Security — View Scan Results** to review detected threats and remediation results. The app displays an error if Defender is disabled or the scan command fails.

During a repair, **Cancel Scan and Repair** asks for confirmation, stops the running DISM or SFC command, and prevents the next step from starting. Changes already applied are not rolled back.

## Automatic Shutdown After Repair

Before starting a repair, select **Shut Down the Computer 5 Minutes After Scan and Repair Complete** and confirm your choice. Windows schedules shutdown only after both steps succeed. The option is off by default and cannot be enabled or changed during a repair, although the cancellation button can disarm it. Failed or cancelled repairs do not schedule shutdown. A scheduling failure is reported separately on the repair completion screen.

The app uses `shutdown.exe /s /t 300`. As described in the [Microsoft documentation](https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/shutdown), a timeout greater than zero implies forced application closure, so unsaved work may be lost. Save your files before using this option. The schedule remains active even if the app is closed.

**Cancel Scheduled Shutdown** runs `shutdown.exe /a` and remains available after reopening the app. Pressing it during a repair also disables automatic shutdown for that repair. This command can cancel a Windows shutdown scheduled by another program or user. If shutdown was already cancelled outside the app, the button also clears the app's pending state.

While the app tracks a pending shutdown, it blocks new scans, repairs, and app update installation. Cancel the shutdown schedule before continuing work.

### Version Comparison and Release Compatibility

This release uses `v2026.10.05`, which the older `2026.09.12` app can recognize. The current app compares three-part and four-part versions numerically, treating an omitted fourth part as zero. Version comparison does not use the computer's date or time. The older `2026.09.12` app recognizes only three-part versions, so it cannot directly update to a four-part release tag.

Build this version with:

```powershell
wails build -clean -platform windows/amd64 -webview2 browser -ldflags "-X main.appVersion=2026.10.05"
```

Use `v2026.10.05` as the stable release tag and attach the compatibility executable named `WindowsSystemRepairHelper.exe`.

## Distribution Filenames

Builds use the executable name configured in `wails.json`. Preserve that filename inside a ZIP for manual downloads because GitHub may normalize non-ASCII release attachment names. For example, name the ZIP `windows-system-repair-helper-2026.10.05.zip`.

For compatibility with automatic updates in existing apps, copy the same executable to `WindowsSystemRepairHelper.exe` and attach it alongside the ZIP. The executable inside the ZIP and the compatibility executable must contain identical bytes. Automatic updates preserve the existing executable path, so they do not rename files already installed on a user's computer.

```powershell
$buildConfig = Get-Content 'wails.json' -Raw | ConvertFrom-Json
$executablePath = Join-Path 'build/bin' ($buildConfig.outputfilename + '.exe')
Copy-Item -LiteralPath $executablePath -Destination 'build/bin/WindowsSystemRepairHelper.exe'
Compress-Archive -LiteralPath $executablePath -DestinationPath 'build/bin/windows-system-repair-helper-2026.10.05.zip' -Force
```

## Distribution Notices

The app's **Developer Information → Licenses and Third-party Notices** screen includes the project license and third-party notices for offline reading. The [notice bundle](THIRD_PARTY_NOTICES.txt) preserves upstream license texts. The [dependency review](legal/DEPENDENCY_REVIEW.md) documents its scope and remaining limitations.

For a release build on Windows, install Python 3 in addition to the development tools above and run:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/build-release.ps1
```

This regenerates notices, builds using the WebView2 browser strategy, checks the EXE's module versions and toolchain against the notices, and packages the executable with `LICENSE` and `THIRD_PARTY_NOTICES.txt`. If the dependency versions or toolchain change, review the generated diff before publishing. Use a new version for a new public release; these source changes do not replace the already published `2026.09.13` assets.

WebView2 includes Microsoft Defender SmartScreen, which collects and sends end-user information to Microsoft as described in the [Microsoft Privacy Statement](https://aka.ms/privacy) and [Microsoft Edge Privacy Whitepaper](https://learn.microsoft.com/en-us/microsoft-edge/privacy-whitepaper#smartscreen). This independent application is not sponsored or endorsed by Microsoft.

### Failure diagnostics

When DISM or SFC exits with an error, the app enables debug mode and captures a snapshot before allowing a retry. Select **Save failure logs** to export a UTF-8 text report containing the execution start/end times (with timezone), failed command, exit error, console output, and DISM/CBS records from that execution window. The report stays in memory until the app closes or another failure replaces it; save it before closing the app.

Collection includes `dism.log`, `dism.log.bak`, `CBS.log`, and up to eight recent `CbsPersist` archives, including CAB files extracted with Windows `expand.exe`. All log severities and continuation lines are retained within the time window. Each source retains its last 2 MiB; truncation, missing files, extraction failures, and empty windows are explicitly reported. CAB extraction has a 15-second timeout per archive. Concurrent Windows servicing activity may appear in the same window. Logs are saved locally and are not uploaded; review paths and user names before sharing.

## Graphics and screen flicker diagnostics

Use **Scan graphics / Recheck after updating** to collect Intel, AMD, and NVIDIA display adapters, their driver versions and device problem codes, active monitor paths, and relevant events from the last seven days. Vendor routing uses PCI hardware IDs rather than CPU names. Hybrid systems show all detected adapters. Unsupported or unavailable information is explicitly reported; a scan with no matching events does not establish that the system is healthy.

**The screen just flickered** records the current time and scans the preceding two minutes. Re-scan shortly afterward to include up to two minutes after the marker. Markers expire after ten minutes, returning scans to the seven-day window. The collector queries Display/vendor/WHEA events, relevant Windows Error Reporting and desktop crashes, and graphics device configuration events. Queries retain up to 200 candidate events per channel and disclose truncation or access failures. Driver dates are package metadata, not installation timestamps. The symptom questions help distinguish app, driver, and external display paths; they do not diagnose an Intel CPU/iGPU fault from its presence alone.

The vendor update button opens Intel's browser-based Driver & Support Assistant, or a signature-verified installed NVIDIA App / AMD Software. If a supported installed tool is not found, it opens the official vendor page. Installation and any restart happen through the vendor's interface; the app does not claim that opening a tool installs a driver. PC manufacturer support links are also available. Before launching, the app saves adapter versions to `%APPDATA%/windows-system-repair-helper/graphics-baseline.json`; later scans compare this baseline even after a restart. Version changes are evidence of a changed driver, not proof that flicker has stopped. The restart indication checks Windows servicing/update restart flags and is not a complete GPU installer status.

**Save graphics diagnostic report** exports the scan, raw evidence, collection limits, version changes, and symptom answers as JSON. Reports stay local; they can contain hardware identifiers, machine model, paths, and event details.

## Release automation

Pushes to `main` run `.github/workflows/release.yml` on Windows. The workflow runs frontend/backend tests, a PowerShell collector fixture, builds with Go 1.27.1 and Wails 2.14.0, verifies license notices, and publishes a new versioned GitHub release containing `WindowsSystemRepairHelper.exe` and a ZIP with license files. Bump `appVersion`, Windows version resources, and `RELEASE_NOTES.md` for each release. Existing releases are not overwritten. The current release version is `2026.10.05`.

### 복구 원본 부족 오류 자동 대응

`DISM /Online /Cleanup-Image /RestoreHealth`가 `0x800f0915` 또는 `0x800f081f`로 실패하면 연결된 드라이브의 `sources/install.wim`·`install.esd`를 자동 검색합니다. Windows 빌드, 에디션, 아키텍처, 설치된 UI 언어가 일치하고 업데이트 리비전이 같거나 높은 이미지의 인덱스를 선택해 `/Source`를 지정한 복구를 한 번 재시도합니다. Windows Update도 보조 원본으로 계속 사용하며, DISM이 성공한 경우에만 SFC로 진행합니다.

자동 복구가 불가능하면 실패 화면의 **설치 원본으로 복구 계속**에서 WIM/ESD 파일을 선택할 수 있습니다. Microsoft 설치 ISO는 Windows에서 먼저 탑재하고 `sources` 폴더의 파일을 선택합니다. 원본 선택 재시도에서는 자동 종료를 예약하지 않습니다. 원본 파일 경로는 명령 셸을 거치지 않고 DISM에 전달합니다.

호환 원본이 없다면 **Windows 복구 재설치 열기**로 복구 설정에 진입할 수 있습니다. Windows의 **Windows 업데이트를 사용하여 문제 해결 → 지금 다시 설치**는 사용자가 Windows 화면에서 시작해야 하며, 지원되지 않거나 관리 정책에 의해 제공되지 않는 PC도 있습니다. 앱이 설정을 여는 것을 복구 완료로 처리하지 않습니다.

이 기능은 ISO 자동 다운로드·탑재 또는 무인 Windows 재설치를 수행하지 않습니다. 서로 다른 빌드 간 enablement package 관계를 추정하지 않으므로, 해당 관계로 호환될 수 있는 미디어도 자동 선택에서 제외합니다. 호환 이미지라도 필요한 파일이 빠져 있으면 복구가 실패할 수 있습니다. 특히 Insider 빌드에 일반 배포 ISO를 무조건 적용하지 않습니다.

근거: [Microsoft 복구 원본 구성](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/configure-a-windows-repair-source?view=windows-11), [Windows 복구 재설치](https://support.microsoft.com/en-US/Windows/Deployment/Install-Upgrade/fix-issues-by-reinstalling-the-current-version-of-windows).


원본 자동 검색에서 후보가 제외되면 **원본 검사 결과**에 대상 Windows와 후보 이미지 버전, 빌드·에디션·아키텍처·언어·업데이트 수준의 제외 사유 또는 조회 오류를 표시합니다. 자동 검색 실패의 상세 내용은 실패 로그에도 포함됩니다. 이 검사는 보수적인 후보 필터이며, 통과한 이미지에 실제 필요한 모든 복구 파일이 있다는 보장은 아닙니다. 다른 빌드의 stable ISO를 자동 대체 원본으로 사용하지 않습니다.

원본을 확보하지 못하거나 DISM 원본 복구가 실패하면 **Windows 복구 재설치 열기**에서 다음 복구 방법을 안내합니다. 이 버튼은 설정만 열며 재설치를 실행하거나 복구 성공으로 처리하지 않습니다. Windows의 ‘Windows 업데이트를 사용하여 문제 해결’에서 직접 시작해야 하며, 관리 정책이나 Windows 버전에 따라 해당 옵션이 없을 수 있습니다. 옵션이 없으면 Microsoft 설치 미디어를 통한 복구 설치를 검토하고, 개인 파일 및 앱 유지 가능 여부를 설치 프로그램에서 확인합니다. 재설치와 재시작 후 앱을 다시 열어 검사 및 복구를 실행해 결과를 확인하세요.
