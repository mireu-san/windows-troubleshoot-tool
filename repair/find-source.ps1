$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Console]::OutputEncoding = New-Object System.Text.UTF8Encoding($false)
$selected = [Environment]::GetEnvironmentVariable('REPAIR_SELECTED_IMAGE')
try {
    $cv = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion'
    $nativeArchitecture = [Environment]::GetEnvironmentVariable('PROCESSOR_ARCHITEW6432')
    if (!$nativeArchitecture) { $nativeArchitecture = [Environment]::GetEnvironmentVariable('PROCESSOR_ARCHITECTURE') }
    $arch = switch ($nativeArchitecture) {
        'AMD64' { 9 }; 'ARM64' { 12 }; 'x86' { 0 }; default { throw 'Unsupported architecture' }
    }
    $languages = @(Get-ChildItem 'HKLM:\SYSTEM\CurrentControlSet\Control\MUI\UILanguages' | ForEach-Object { $_.PSChildName })
    $current = @{
        Edition = [string]$cv.EditionID; Architecture = $arch
        Major = [int]$cv.CurrentMajorVersionNumber; Minor = [int]$cv.CurrentMinorVersionNumber
        Build = [int]$cv.CurrentBuildNumber; Revision = [int]$cv.UBR; Languages = $languages
    }
    $paths = @()
    if ($selected) {
        if ([IO.Path]::GetExtension($selected) -notin @('.wim', '.esd')) { throw 'Select an install.wim or install.esd file' }
        $paths = @((Get-Item -LiteralPath $selected).FullName)
    } else {
        # Only standard installation-media locations. Do not recursively scan user files or networks.
        foreach ($drive in [IO.DriveInfo]::GetDrives()) {
            if (!$drive.IsReady -or $drive.DriveType -notin @('CDRom', 'Removable', 'Fixed')) { continue }
            foreach ($name in @('install.wim', 'install.esd')) {
                $path = Join-Path $drive.RootDirectory.FullName "sources\$name"
                if (Test-Path -LiteralPath $path -PathType Leaf) { $paths += $path }
            }
        }
    }
    $images = @()
    $issues = @()
    foreach ($path in $paths) {
        try {
            foreach ($entry in @(Get-WindowsImage -ImagePath $path)) {
                $detail = Get-WindowsImage -ImagePath $path -Index $entry.ImageIndex
                $version = [version]$detail.Version
                $images += @{
                    Path = $path; Index = [int]$detail.ImageIndex; Edition = [string]$detail.EditionId
                    Architecture = [int]$detail.Architecture
                    Major = $version.Major; Minor = $version.Minor; Build = $version.Build; Revision = $version.Revision
                    Languages = @($detail.Languages | ForEach-Object { [string]$_ })
                }
            }
        } catch { $issues += ($path + ': ' + $_.Exception.Message) }
    }
    @{ Current = $current; Images = $images; Issues = $issues } | ConvertTo-Json -Depth 6 -Compress
} catch {
    [Console]::Error.WriteLine($_.Exception.Message)
    exit 1
}
