$ErrorActionPreference = 'Stop'
$script = Get-Content (Join-Path $PSScriptRoot '../graphics/comprehensive.ps1') -Raw
$tokens = $null; $parseErrors = $null
[void][System.Management.Automation.Language.Parser]::ParseInput($script, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
function Get-CimInstance {
 param($ClassName,$Namespace)
 switch ($ClassName) {
  'Win32_Processor' { [pscustomobject]@{Name='Test CPU';Manufacturer='CPU maker';NumberOfCores=8;NumberOfLogicalProcessors=16;Status='OK'} }
  'Win32_OperatingSystem' { [pscustomobject]@{Caption='Windows';Version='10.0';BuildNumber='123'} }
  'AntiVirusProduct' { throw 'SecurityCenter unavailable' }
  'Win32_StartupCommand' { [pscustomobject]@{Name='Test startup';Location='Registry';Command='secret command'} }
 }
}
function Get-NetFirewallProfile {
 param($PolicyStore)
 if ($PolicyStore -ne 'ActiveStore') { throw 'Must inspect effective policy' }
 [pscustomobject]@{Name='Public';Enabled=$false}
 [pscustomobject]@{Name='Private';Enabled=$true}
}
function Get-NetConnectionProfile { [pscustomobject]@{InterfaceAlias='Ethernet';NetworkCategory='Public';IPv4Connectivity='Internet'} }
function Get-MpComputerStatus { throw 'Access denied' }
function Get-Process { [pscustomobject]@{ProcessName='AnyDesk';Id=42;StartTime=[datetime]'2026-10-05'} }
function Get-ScheduledTask { 1..205 | ForEach-Object { [pscustomobject]@{TaskName="Task $_";TaskPath='\Custom\';State='Ready'} } }
$script:denySecurity = $false
function Get-WinEvent {
 param($LogName,$FilterXPath,$MaxEvents)
 if ($FilterXPath -notmatch 'TimeCreated' -or $MaxEvents -ne 201) { throw 'Query is not bounded' }
 if ($script:denySecurity -and $LogName -eq 'Security') { throw 'Security access denied' }
 $row = [pscustomobject]@{TimeCreated=[datetime]'2026-10-05T12:00:00';ProviderName='Fixture';Id=4688;RecordId=12;Message='message with secret command'}
 $row | Add-Member ScriptMethod ToXml { '<Event><EventData><Data Name="NewProcessName">C:\game.exe</Data><Data Name="CommandLine">secret command</Data></EventData></Event>' }
 $row
}
$script=$script.Replace('__START__','2026-10-01T00:00:00Z').Replace('__END__','2026-10-06T00:00:00Z')
$result = & ([scriptblock]::Create($script)) | ConvertFrom-Json
if ($result.cpu.Count -ne 1 -or $result.cpu[0].NumberOfCores -ne 8) { throw 'CPU inventory lost' }
if ($result.firewall[0].enabled -ne 'False' -or $result.network[0].category -ne 'Public') { throw 'Firewall/network state lost' }
if ($result.defender.Count -ne 0 -or ($result.issues -join ';') -notmatch 'Access denied') { throw 'Missing protection state must be disclosed' }
if ($result.tasks.Count -ne 200 -or ($result.issues -join ';') -notmatch 'truncated') { throw 'Task bound not enforced' }
if ($result.processEvents[0].data.NewProcessName -ne 'C:\game.exe' -or $result.processEvents[0].data.CommandLine -or $result.processEvents[0].message) { throw 'Process data missing or command line leaked' }
if ($result.startup[0].Command) { throw 'Startup command leaked' }
if ($result.remoteProcesses.Count -ne 1) { throw 'Remote tool clue lost' }
$script:denySecurity = $true
$result = & ([scriptblock]::Create($script)) | ConvertFrom-Json
if ($result.processEvents.Count -ne 0 -or ($result.issues -join ';') -notmatch 'Security access denied' -or $result.cpu.Count -ne 1) { throw 'Partial failure discarded other evidence' }
Write-Host 'Comprehensive flicker collector fixtures passed.'
