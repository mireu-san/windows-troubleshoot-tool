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

The current default version is `2026.09.15`. Specify the version when building a release on Windows:

```powershell
wails build -clean -platform windows/amd64 -webview2 browser -ldflags "-X main.appVersion=2026.09.15"
```

1. Commit the changes and push them to GitHub.
2. Create a stable release with a tag such as `v2026.09.15`. The tag must match the version embedded in the build. Supported formats are `vYYYY.MM.DD` and `vYYYY.MM.DD.N`; increment the last number for additional releases on the same day. See the compatibility notes below before adopting a four-part tag.
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

This release uses `v2026.09.15`, which the older `2026.09.12` app can recognize. The current app compares three-part and four-part versions numerically, treating an omitted fourth part as zero. Version comparison does not use the computer's date or time. The older `2026.09.12` app recognizes only three-part versions, so it cannot directly update to a four-part release tag.

Build this version with:

```powershell
wails build -clean -platform windows/amd64 -webview2 browser -ldflags "-X main.appVersion=2026.09.15"
```

Use `v2026.09.15` as the stable release tag and attach the compatibility executable named `WindowsSystemRepairHelper.exe`.

## Distribution Filenames

Builds use the executable name configured in `wails.json`. Preserve that filename inside a ZIP for manual downloads because GitHub may normalize non-ASCII release attachment names. For example, name the ZIP `windows-system-repair-helper-2026.09.15.zip`.

For compatibility with automatic updates in existing apps, copy the same executable to `WindowsSystemRepairHelper.exe` and attach it alongside the ZIP. The executable inside the ZIP and the compatibility executable must contain identical bytes. Automatic updates preserve the existing executable path, so they do not rename files already installed on a user's computer.

```powershell
$buildConfig = Get-Content 'wails.json' -Raw | ConvertFrom-Json
$executablePath = Join-Path 'build/bin' ($buildConfig.outputfilename + '.exe')
Copy-Item -LiteralPath $executablePath -Destination 'build/bin/WindowsSystemRepairHelper.exe'
Compress-Archive -LiteralPath $executablePath -DestinationPath 'build/bin/windows-system-repair-helper-2026.09.15.zip' -Force
```

## Distribution Notices

The app's **Developer Information → Licenses and Third-party Notices** screen includes the project license and third-party notices for offline reading. The [notice bundle](THIRD_PARTY_NOTICES.txt) preserves upstream license texts. The [dependency review](legal/DEPENDENCY_REVIEW.md) documents its scope and remaining limitations.

For a release build on Windows, install Python 3 in addition to the development tools above and run:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/build-release.ps1
```

This regenerates notices, builds using the WebView2 browser strategy, checks the EXE's module versions and toolchain against the notices, and packages the executable with `LICENSE` and `THIRD_PARTY_NOTICES.txt`. If the dependency versions or toolchain change, review the generated diff before publishing. Use a new version for a new public release; these source changes do not replace the already published `2026.09.13` assets.

WebView2 includes Microsoft Defender SmartScreen, which collects and sends end-user information to Microsoft as described in the [Microsoft Privacy Statement](https://aka.ms/privacy) and [Microsoft Edge Privacy Whitepaper](https://learn.microsoft.com/en-us/microsoft-edge/privacy-whitepaper#smartscreen). This independent application is not sponsored or endorsed by Microsoft.
