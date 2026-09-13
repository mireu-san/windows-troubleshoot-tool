# Run from Windows with Go, Node.js, Python 3 and Wails CLI installed.
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
try {
    npm --prefix frontend ci
    if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency installation failed.' }
    npm --prefix frontend test
    if ($LASTEXITCODE -ne 0) { throw 'Frontend tests failed.' }
    npm --prefix frontend run build
    if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed.' }
    python scripts/generate_notices.py
    if ($LASTEXITCODE -ne 0) { throw 'Notice generation failed.' }
    wails build -clean -platform windows/amd64 -webview2 browser
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed.' }
    $buildConfig = Get-Content 'wails.json' -Raw | ConvertFrom-Json
    $executablePath = Join-Path 'build/bin' ($buildConfig.outputfilename + '.exe')
    python scripts/generate_notices.py --check --binary $executablePath
    if ($LASTEXITCODE -ne 0) { throw 'EXE notice verification failed.' }
    Copy-Item -LiteralPath $executablePath -Destination 'build/bin/WindowsSystemRepairHelper.exe'
    Copy-Item 'LICENSE', 'THIRD_PARTY_NOTICES.txt' -Destination 'build/bin/'
    $versionMatch = [regex]::Match((Get-Content 'app_update.go' -Raw), 'var appVersion = "([0-9.]+)"')
    if (-not $versionMatch.Success) { throw 'Cannot read the app version.' }
    $archivePath = 'build/bin/windows-system-repair-helper-' + $versionMatch.Groups[1].Value + '.zip'
    Compress-Archive -LiteralPath $executablePath, 'build/bin/LICENSE', 'build/bin/THIRD_PARTY_NOTICES.txt' -DestinationPath $archivePath -Force
    Write-Host "Ready for review: $archivePath"
} finally {
    Pop-Location
}
