$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$from = [DateTimeOffset]::Parse('__START__').LocalDateTime
$until = [DateTimeOffset]::Parse('__END__').LocalDateTime
$issues = [System.Collections.Generic.List[string]]::new()
$gpus = @()
$events = [System.Collections.Generic.List[object]]::new()
$pc = $null
$pending = $false
try { $pc = Get-CimInstance Win32_ComputerSystem | Select-Object Manufacturer,Model } catch { $issues.Add('Computer: ' + $_.Exception.Message) }
try {
 $signed = @(Get-CimInstance Win32_PnPSignedDriver -Filter "DeviceClass='DISPLAY'")
} catch { $signed = @(); $issues.Add('Driver metadata: ' + $_.Exception.Message) }
try {
 $gpus = @(Get-CimInstance Win32_VideoController | ForEach-Object {
  $v = $_
  $d = $signed | Where-Object { $_.DeviceID -eq $v.PNPDeviceID } | Select-Object -First 1
  [pscustomobject]@{
   id = [string]$v.PNPDeviceID; name = [string]$v.Name
   version = [string]$v.DriverVersion; provider = [string]$d.DriverProviderName
   driverDate = $(if ($v.DriverDate) { $v.DriverDate.ToString('o') } else { '' })
   inf = [string]$d.InfName; signed = $(if ($d) { [bool]$d.IsSigned } else { $null })
   problemCode = $v.ConfigManagerErrorCode; status = [string]$v.Status
  }
 })
} catch { $issues.Add('GPU inventory: ' + $_.Exception.Message) }
# A bounded, time-filtered query per channel. XML fields avoid localized message matching.
$startUTC = $from.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
$endUTC = $until.ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
$timeFilter = "TimeCreated[@SystemTime>='$startUTC' and @SystemTime<='$endUTC']"
$queries = @(
 @{ log='System'; condition="(Provider[@Name='Display'] or Provider[@Name='nvlddmkm'] or Provider[@Name='amdkmdag'] or Provider[@Name='amdwddmg'] or Provider[@Name='igfx'] or Provider[@Name='igfxn'] or Provider[@Name='igdkmdn64'] or Provider[@Name='Microsoft-Windows-WHEA-Logger']) and (Level=1 or Level=2 or Level=3)" },
 @{ log='Application'; condition="(Provider[@Name='Windows Error Reporting'] and EventID=1001) or (Provider[@Name='Application Error'] and EventID=1000)" },
 @{ log='Microsoft-Windows-Kernel-PnP/Configuration'; condition="EventID=400 or EventID=410 or EventID=411" }
)
foreach ($query in $queries) {
 try {
  $rows = @(Get-WinEvent -LogName $query.log -FilterXPath "*[System[$timeFilter and ($($query.condition))]]" -MaxEvents 201)
  if ($rows.Count -gt 200) { $issues.Add($query.log + ': truncated to 200 newest candidate events') }
  foreach ($row in ($rows | Select-Object -First 200)) {
   $xml = [xml]$row.ToXml()
   $data = @{}
   foreach ($item in $xml.Event.EventData.Data) { if ($item.Name) { $data[[string]$item.Name] = [string]$item.'#text' } }
   $keep = $query.log -eq 'System'
   if ($query.log -eq 'Application') { $keep = ($data.EventName -eq 'LiveKernelEvent') -or ($data.AppName -match '^(dwm|explorer)\.exe$') }
   if ($query.log -like '*Kernel-PnP*') { $keep = @($gpus | Where-Object { $_.id -eq $data.DeviceInstanceId }).Count -gt 0 }
   if ($keep) {
    $message = [string]$row.Message
    if ($message.Length -gt 6000) { $message = $message.Substring(0,6000); $issues.Add('Event message truncated: ' + $row.RecordId) }
    $events.Add([pscustomobject]@{ time=$row.TimeCreated.ToString('o'); provider=$row.ProviderName; id=$row.Id; recordId=$row.RecordId; log=$query.log; message=$message; data=$data })
   }
  }
 } catch { if ($_.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*') { $issues.Add($query.log + ': ' + $_.Exception.Message) } }
}
try {
 $pending = (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending') -or (Test-Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired')
} catch { $issues.Add('Restart status: ' + $_.Exception.Message) }
[pscustomobject]@{ gpus=@($gpus); events=@($events.ToArray()); issues=@($issues.ToArray()); manufacturer=[string]$pc.Manufacturer; model=[string]$pc.Model; restartPending=$pending } | ConvertTo-Json -Depth 8 -Compress
