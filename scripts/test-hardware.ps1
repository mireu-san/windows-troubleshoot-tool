$ErrorActionPreference = 'Stop'
$script = Get-Content (Join-Path $PSScriptRoot '../graphics/hardware.ps1') -Raw -Encoding UTF8
$tokens=$null; $parseErrors=$null
[void][System.Management.Automation.Language.Parser]::ParseInput($script,[ref]$tokens,[ref]$parseErrors)
if ($parseErrors.Count) {throw ($parseErrors | Out-String)}
function Get-CimInstance {
 param($ClassName,$Namespace,$Filter)
 switch ($ClassName) {
  'Win32_Processor' {[pscustomobject]@{Name='Fixture CPU';Manufacturer='Intel'}}
  'Win32_BaseBoard' {throw 'Access denied fixture'}
  'Win32_PhysicalMemory' {[pscustomobject]@{Capacity=8589934592;SMBIOSMemoryType=26;ConfiguredClockSpeed=2666}}
 }
}
function Get-PhysicalDisk {return}
function Get-StorageReliabilityCounter {return}
function Test-Path {param($LiteralPath); return $false}
function Get-WinEvent {
 param($LogName,$FilterXPath,$MaxEvents)
 if ($FilterXPath -notmatch 'TimeCreated' -or $MaxEvents -ne 1001) {throw 'Unbounded query'}
 if ($FilterXPath -notmatch "Provider\[@Name='Display'\]") {return}
 $row=[pscustomobject]@{TimeCreated=[datetime]'2026-10-08T01:00:00Z';ProviderName='Display';Id=4101;RecordId=1;LevelDisplayName='Warning';Message='SECRET user full path'}
 $row | Add-Member ScriptMethod ToXml {'<Event><EventData><Data Name="UserName">SECRET</Data><Data Name="EventName">LiveKernelEvent</Data><Data Name="P1">141</Data></EventData></Event>'}
 if ($script:HardwareFixtureMany) { for ($i=0; $i -lt 1001; $i++) { $row } } else { $row }
}
$env:ProgramData = $PSScriptRoot
$env:SystemRoot = $PSScriptRoot
$script=$script.Replace('__START__','2026-09-08T01:00:00Z').Replace('__END__','2026-10-08T02:00:00Z').Replace('__INCIDENT__','2026-10-08T01:00:00Z')
$raw=& ([scriptblock]::Create($script))
$result=$raw | ConvertFrom-Json
if ($result.events.Count -ne 1) {throw 'Event deduplication failed'}
if ($raw -match 'SECRET') {throw 'Private event fields escaped allowlist'}
if ($result.events[0].description -notmatch '141') {throw 'LiveKernel code missing'}
if ($result.sections.CPU.data[0].Name -ne 'Fixture CPU') {throw 'CPU lost'}
$failed=@($result.sections.PSObject.Properties | Where-Object {$_.Value.status -match 'Access denied|조회 실패'})
if (!$failed.Count) {throw 'Partial collection not disclosed'}
$script:HardwareFixtureMany=$true
$result = (& ([scriptblock]::Create($script))) | ConvertFrom-Json
$limited=@($result.sections.PSObject.Properties | Where-Object {$_.Value.status -match '1000'})
if (!$limited.Count) {throw 'Candidate truncation not disclosed'}
Write-Host 'Hardware collector fixture passed.'
