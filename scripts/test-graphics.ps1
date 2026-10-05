$ErrorActionPreference = 'Stop'
$script = Get-Content (Join-Path $PSScriptRoot '../graphics/scan.ps1') -Raw
$tokens = $null
$parseErrors = $null
[void][System.Management.Automation.Language.Parser]::ParseInput($script, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
# Exercise the actual collector with deterministic CIM and event-log fixtures.
function Get-CimInstance {
 param($ClassName, $Filter)
 switch ($ClassName) {
  'Win32_ComputerSystem' { [pscustomobject]@{Manufacturer='Example OEM';Model='Hybrid laptop'} }
  'Win32_PnPSignedDriver' { [pscustomobject]@{DeviceID='PCI\VEN_8086&DEV_1234';DriverProviderName='Intel';InfName='oem1.inf';IsSigned=$true} }
  'Win32_VideoController' { [pscustomobject]@{PNPDeviceID='PCI\VEN_8086&DEV_1234';Name='Intel Graphics';DriverVersion='1.2.3';DriverDate=[datetime]'2026-09-01';ConfigManagerErrorCode=0;Status='OK'} }
 }
}
function Get-WinEvent {
 param($LogName,$FilterXPath,$MaxEvents)
 if ($FilterXPath -notmatch 'TimeCreated' -or $MaxEvents -ne 201) { throw 'Query must be bounded and time-filtered' }
 if ($LogName -ne 'System') { return }
 $row = [pscustomobject]@{TimeCreated=[datetime]'2026-10-05T12:00:00';ProviderName='Display';Id=4101;RecordId=123;Message='Display driver recovered'}
 $row | Add-Member ScriptMethod ToXml { '<Event><EventData><Data Name="param1">igfx</Data></EventData></Event>' }
 $row
}
$script=$script.Replace('__START__','2026-10-05T00:00:00Z').Replace('__END__','2026-10-06T00:00:00Z')
$result = & ([scriptblock]::Create($script)) | ConvertFrom-Json
if ($result.gpus.Count -ne 1 -or $result.gpus[0].provider -ne 'Intel' -or $result.events[0].data.param1 -ne 'igfx') { throw 'Collector lost device or event metadata' }
Write-Host 'Graphics collector fixture passed.'
