$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$result = [ordered]@{}
$issues = [System.Collections.Generic.List[string]]::new()
function Collect($name, [scriptblock]$query) {
 try { $result[$name] = @(& $query) } catch { $result[$name] = @(); $issues.Add($name + ': ' + $_.Exception.Message) }
}
Collect 'cpu' { Get-CimInstance Win32_Processor | Select-Object Name,Manufacturer,NumberOfCores,NumberOfLogicalProcessors,Status }
Collect 'windows' { Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,BuildNumber }
Collect 'firewall' { Get-NetFirewallProfile -PolicyStore ActiveStore | ForEach-Object { [pscustomobject]@{name=[string]$_.Name; enabled=[string]$_.Enabled} } }
Collect 'network' { Get-NetConnectionProfile | Select-Object InterfaceAlias,@{n='category';e={[string]$_.NetworkCategory}},IPv4Connectivity,IPv6Connectivity }
Collect 'defender' { Get-MpComputerStatus | Select-Object AMRunningMode,AntivirusEnabled,RealTimeProtectionEnabled,AntivirusSignatureLastUpdated }
Collect 'antivirus' { Get-CimInstance -Namespace root/SecurityCenter2 -ClassName AntiVirusProduct | Select-Object displayName,productState }
Collect 'remoteProcesses' { Get-Process | Where-Object { $_.ProcessName -match '^(AnyDesk|TeamViewer.*|RustDesk|mstsc|msrdc|ScreenConnect.*|winvnc|tvnserver|parsecd|sunshine|remoting_host)$' } | Select-Object ProcessName,Id,StartTime }
Collect 'startup' { Get-CimInstance Win32_StartupCommand | Select-Object -First 201 Name,Location }
Collect 'tasks' { Get-ScheduledTask | Where-Object { $_.TaskPath -notlike '\Microsoft\*' } | Select-Object -First 201 TaskName,TaskPath,@{n='state';e={[string]$_.State}} }
foreach ($key in @('startup','tasks')) {
 if ($result[$key].Count -gt 200) { $result[$key] = @($result[$key] | Select-Object -First 200); $issues.Add($key + ': truncated to 200 entries') }
}
$start = [DateTimeOffset]::Parse('__START__').UtcDateTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
$end = [DateTimeOffset]::Parse('__END__').UtcDateTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
$queries = @(
 @{name='remoteEvents';log='Microsoft-Windows-TerminalServices-LocalSessionManager/Operational';condition='EventID=21 or EventID=24 or EventID=25'},
 @{name='driverEvents';log='System';condition="Provider[@Name='Microsoft-Windows-Kernel-PnP'] or Provider[@Name='Microsoft-Windows-UserPnp'] or Provider[@Name='Microsoft-Windows-WindowsUpdateClient']"},
 @{name='processEvents';log='Security';condition='EventID=4688'}
)
foreach ($query in $queries) {
 try {
  $rows = @(Get-WinEvent -LogName $query.log -FilterXPath "*[System[TimeCreated[@SystemTime>='$start' and @SystemTime<='$end'] and ($($query.condition))]]" -MaxEvents 201)
  if ($rows.Count -gt 200) { $issues.Add($query.name + ': truncated to 200 newest events') }
  $result[$query.name] = @($rows | Select-Object -First 200 | ForEach-Object {
   $row = $_; $data = @{}; $xml = [xml]$row.ToXml()
   foreach ($item in $xml.Event.EventData.Data) {
    # Do not export process command lines; they can contain credentials.
    if ($item.Name -and $item.Name -ne 'CommandLine') { $data[[string]$item.Name] = [string]$item.'#text' }
   }
   $message = ''
   if ($query.name -ne 'processEvents') {
    $message = [string]$row.Message
    if ($message.Length -gt 6000) { $message = $message.Substring(0,6000); $issues.Add('Event message truncated: ' + $row.RecordId) }
   }
   [pscustomobject]@{time=$row.TimeCreated.ToString('o');provider=$row.ProviderName;id=$row.Id;recordId=$row.RecordId;log=$query.log;message=$message;data=$data}
  })
 } catch {
  $result[$query.name] = @()
  if ($_.FullyQualifiedErrorId -notlike 'NoMatchingEventsFound*') { $issues.Add($query.name + ': ' + $_.Exception.Message) }
 }
}
$issues.Add('Process creation history requires pre-existing auditing and Security log access. Empty results do not prove no activity.')
$issues.Add('Remote process matching is a limited name list, not malware detection. Startup/tasks are inventory, not a verdict. Command lines are omitted.')
$issues.Add('HDR/VRR state and physical cable/port/power faults are not collected. Integrated/discrete GPU classification is unconfirmed.')
$result['issues'] = @($issues.ToArray())
$result | ConvertTo-Json -Depth 8 -Compress
