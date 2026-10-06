# Read-only scanner contract test; stubs Windows metadata and DISM image inspection.
$ErrorActionPreference = 'Stop'
$scriptPath = Join-Path $PSScriptRoot '../repair/find-source.ps1'
$tokens = $null
$parseErrors = $null
$null = [System.Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
$sample = Join-Path ([IO.Path]::GetTempPath()) ('repair & image-' + [guid]::NewGuid() + '.wim')
$previousSelected = $env:REPAIR_SELECTED_IMAGE
$previousArch = $env:PROCESSOR_ARCHITECTURE
$previousNativeArch = $env:PROCESSOR_ARCHITEW6432
try {
    Set-Content -LiteralPath $sample -Value 'test fixture, not a real image'
    $env:REPAIR_SELECTED_IMAGE = $sample
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $env:PROCESSOR_ARCHITEW6432 = $null
    function Get-ItemProperty {
        param($Path)
        [pscustomobject]@{EditionID='Professional'; CurrentMajorVersionNumber=10; CurrentMinorVersionNumber=0; CurrentBuildNumber='26300'; UBR=9550}
    }
    function Get-ChildItem {
        param($Path)
        [pscustomobject]@{PSChildName='ko-KR'}
    }
    function Get-WindowsImage {
        param($ImagePath, $Index)
        if ($ImagePath -ne $sample) { throw 'Image path was modified' }
        if (!$Index) { return [pscustomobject]@{ImageIndex=6} }
        [pscustomobject]@{ImageIndex=6;EditionId='Professional';Architecture=9;Version=[version]'10.0.26300.9550';Languages=@('ko-KR')}
    }
    $result = (& $scriptPath) | ConvertFrom-Json
    if ($result.Current.Build -ne 26300 -or $result.Images.Count -ne 1 -or $result.Images[0].Index -ne 6 -or $result.Images[0].Path -ne $sample -or $result.Images[0].Languages[0] -ne 'ko-KR') { throw 'Scanner result mismatch' }
    function Get-WindowsImage { throw 'unreadable test image' }
    $result = (& $scriptPath) | ConvertFrom-Json
    if ($result.Images.Count -ne 0 -or $result.Issues.Count -ne 1) { throw 'Unreadable image was not reported' }
    Write-Host 'Repair source scanner contract tests passed.'
} finally {
    $env:REPAIR_SELECTED_IMAGE = $previousSelected
    $env:PROCESSOR_ARCHITECTURE = $previousArch
    $env:PROCESSOR_ARCHITEW6432 = $previousNativeArch
    Remove-Item -LiteralPath $sample -Force -ErrorAction SilentlyContinue
}
