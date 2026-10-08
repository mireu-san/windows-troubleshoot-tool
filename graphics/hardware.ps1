$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
$from = [DateTimeOffset]::Parse('__START__')
$until = [DateTimeOffset]::Parse('__END__')
$sections = [ordered]@{}
$events = [System.Collections.Generic.List[object]]::new()
$issues = [System.Collections.Generic.List[string]]::new()
function Collect($name, [scriptblock]$query) {
 try {
  $rows = @(& $query)
  $status = '수집 완료'
  if ($rows.Count -eq 0) { $status = '조회 완료 / 기록 없음 또는 미노출' }
  $sections[$name] = @{status=$status;data=$rows}
 } catch { $sections[$name] = @{status=('조회 실패 / 미지원 또는 권한 부족: ' + $_.FullyQualifiedErrorId);data=@()} }
}
Collect 'CPU' { Get-CimInstance Win32_Processor | Select-Object Name,Manufacturer,SocketDesignation,NumberOfCores,NumberOfLogicalProcessors }
Collect '메인보드' { Get-CimInstance Win32_BaseBoard | Select-Object Manufacturer,Product,Version }
Collect 'BIOS' { Get-CimInstance Win32_BIOS | Select-Object Manufacturer,SMBIOSBIOSVersion,@{n='ReleaseDate';e={if ($_.ReleaseDate) {$_.ReleaseDate.ToString('o')}}} }
Collect 'RAM 모듈' { Get-CimInstance Win32_PhysicalMemory | Select-Object DeviceLocator,BankLabel,Capacity,Speed,ConfiguredClockSpeed,SMBIOSMemoryType,Manufacturer,PartNumber }
Collect 'RAM 총량' { Get-CimInstance Win32_ComputerSystem | Select-Object TotalPhysicalMemory }
Collect 'Windows' { Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,BuildNumber,OSArchitecture }
Collect 'GPU' { Get-CimInstance Win32_VideoController | Select-Object Name,DriverVersion,@{n='DriverDate';e={if ($_.DriverDate) {$_.DriverDate.ToString('o')}}},ConfigManagerErrorCode,Status }
Collect '그래픽 드라이버' { Get-CimInstance Win32_PnPSignedDriver -Filter "DeviceClass='DISPLAY'" | Select-Object DeviceName,DriverVersion,DriverProviderName,InfName,IsSigned }
function Decode($bytes) { -join @($bytes | Where-Object {$_ -ne 0} | ForEach-Object {[char]$_}) }
Collect '모니터 모델' { Get-CimInstance -Namespace root/wmi -ClassName WmiMonitorID | ForEach-Object { [pscustomobject]@{Manufacturer=(Decode $_.ManufacturerName);Model=(Decode $_.UserFriendlyName);Product=(Decode $_.ProductCodeID);Active=$_.Active} } }
Collect '논리적 모니터 출력' { Get-CimInstance -Namespace root/wmi -ClassName WmiMonitorConnectionParams | Select-Object Active,VideoOutputTechnology }
Collect '저장장치' { Get-CimInstance Win32_DiskDrive | Select-Object Model,InterfaceType,Size,Status }
Collect '저장장치 건강' { Get-PhysicalDisk | Select-Object FriendlyName,MediaType,BusType,HealthStatus,OperationalStatus,Size }
Collect 'SMART 예측' { Get-CimInstance -Namespace root/wmi -ClassName MSStorageDriver_FailurePredictStatus | Select-Object PredictFailure,Reason }
Collect '저장장치 신뢰성' { Get-PhysicalDisk | Get-StorageReliabilityCounter | Select-Object DeviceId,Temperature,Wear,PowerOnHours,ReadErrorsTotal,WriteErrorsTotal }
$issues.Add('RAM Capacity/TotalPhysicalMemory/Size는 바이트, Speed/ConfiguredClockSpeed는 펌웨어 보고값입니다. SMBIOSMemoryType 26=DDR4, 34=DDR5, 기타 값은 별도 확인하세요. GPU 이름은 내장 GPU 판정이 아닙니다. DriverDate는 설치 시각이 아닙니다.')
$issues.Add('VideoOutputTechnology: 0=HD15/VGA, 4=DVI, 5=HDMI, 10=외부 DisplayPort, 11=내장 DisplayPort. Windows 논리 보고값이며 변환기·실제 배선은 확인되지 않습니다.')
$windows = @(@{start=$from;end=$until;label='최근 30일'})
if ('__INCIDENT__' -ne '') {
 $incident=[DateTimeOffset]::Parse('__INCIDENT__')
 $end=$incident.AddMinutes(2); if ($end -gt $until) {$end=$until}
 $windows += @{start=$incident.AddMinutes(-2);end=$end;label='사건 전후 2분'}
}
$queries = @(
 @{name='Display';log='System';condition="Provider[@Name='Display']";relevance='그래픽 드라이버 복구/표시 기록'},
 @{name='GPU';log='System';condition="Provider[@Name='igfx'] or Provider[@Name='igfxn'] or Provider[@Name='igdkmdn64'] or Provider[@Name='nvlddmkm'] or Provider[@Name='amdkmdag'] or Provider[@Name='amdwddmg']";relevance='그래픽 공급자 기록'},
 @{name='WHEA';log='System';condition="Provider[@Name='Microsoft-Windows-WHEA-Logger']";relevance='하드웨어 오류 보고 / 부품 확정 불가'},
 @{name='Power';log='System';condition="(Provider[@Name='Microsoft-Windows-Kernel-Power'] and EventID=41) or (Provider[@Name='EventLog'] and EventID=6008)";relevance='비정상 종료 / PSU 고장 증거 아님'},
 @{name='Boot';log='System';condition="(Provider[@Name='Microsoft-Windows-Kernel-Boot'] or Provider[@Name='Microsoft-Windows-Kernel-PnP']) and (Level=1 or Level=2 or Level=3)";relevance='부팅·장치 오류 / 장치 확인 필요'},
 @{name='WER';log='Application';condition="Provider[@Name='Windows Error Reporting'] and EventID=1001";relevance='그래픽 WER 기록'},
 @{name='Crash';log='Application';condition="Provider[@Name='Application Error'] and EventID=1000";relevance='그래픽 화면 구성 앱 오류'}
)
$seen = @{}
foreach ($window in $windows) {
 $startUTC=$window.start.UtcDateTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ'); $endUTC=$window.end.UtcDateTime.ToString('yyyy-MM-ddTHH:mm:ss.fffZ')
 foreach ($q in $queries) {
  $name=$window.label+' / '+$q.name
  try {
   $rows=@(Get-WinEvent -LogName $q.log -FilterXPath "*[System[TimeCreated[@SystemTime>='$startUTC' and @SystemTime<='$endUTC'] and ($($q.condition))]]" -MaxEvents 1001)
   $count=0
   foreach ($row in ($rows | Select-Object -First 1000)) {
    $xml=[xml]$row.ToXml(); $data=@{}
    foreach ($item in $xml.Event.EventData.Data) {if ($item.Name) {$data[[string]$item.Name]=[string]$item.'#text'}}
    if ($q.name -eq 'WER' -and $data.EventName -ne 'LiveKernelEvent' -and $data.P1 -notmatch '^(dwm|explorer)\.exe$') {continue}
    if ($q.name -eq 'Crash' -and $data.AppName -notmatch '^(dwm|explorer)\.exe$') {continue}
    $count++
    $key=$q.log+':'+$row.RecordId; if ($seen.ContainsKey($key)) {continue};$seen[$key]=$true
    # Never export arbitrary messages, paths, usernames or WER parameters.
    $fields=@()
    foreach ($field in @('EventName','BugcheckCode','ErrorSource','ErrorType','ApicId','MCABank','MciStat')) {
     if ($data[$field] -match '^[A-Za-z0-9_x -]{1,80}$') {$fields+=($field+'='+$data[$field])}
    }
    if ($data.EventName -eq 'LiveKernelEvent' -and $data.P1 -match '^[0-9A-Fa-f]{1,8}$') {$fields+=('LiveKernelEvent code='+$data.P1)}
    $relevance=$q.relevance
    if ($data.EventName -eq 'LiveKernelEvent' -and $data.P1 -notin @('117','141')) {$relevance='커널 오류 / 그래픽 관련성 미확정'}
    $events.Add([pscustomobject]@{time=([DateTimeOffset]$row.TimeCreated).ToString('o');source=$row.ProviderName;id=[string]$row.Id;severity=[string]$row.LevelDisplayName;description=($q.name+'; '+($fields -join '; '));relevance=$relevance})
   }
   $status='수집 완료'; if ($count -eq 0) {$status='조회 완료 / 관련 기록 없음'}; if ($rows.Count -gt 1000) {$status='일부 수집 / 최신 후보 1000건 제한';$issues.Add($name+': 후보 제한으로 관련 사건이 누락될 수 있음')}
   $sections[$name]=@{status=$status;data=@(@{candidates=$rows.Count;relevant=$count})}
  } catch {
   $status='조회 실패: '+$_.FullyQualifiedErrorId
   if ($_.FullyQualifiedErrorId -like 'NoMatchingEventsFound*') {$status='조회 완료 / 해당 기록 없음'}
   $sections[$name]=@{status=$status;data=@()}
  }
 }
}
Collect '신뢰성 기록' {
 Get-CimInstance Win32_ReliabilityRecords -Filter ("TimeGenerated >= '"+$from.UtcDateTime.ToString('yyyyMMddHHmmss')+".000000+000'") | Where-Object {$_.TimeGenerated -ge $from.LocalDateTime -and $_.ProductName -match '^(Windows|Microsoft Windows|dwm\.exe|explorer\.exe)$'} | Select-Object -First 501 | ForEach-Object {
  $stamp=([DateTimeOffset]$_.TimeGenerated).ToString('o')
  $events.Add([pscustomobject]@{time=$stamp;source=('Reliability / '+$_.SourceName);id=[string]$_.EventIdentifier;severity='신뢰성 기록';description=$_.ProductName;relevance='신뢰성 기록 / 이벤트 로그와 중복 가능'})
  [pscustomobject]@{Time=$stamp;Source=$_.SourceName;EventIdentifier=$_.EventIdentifier;Product=$_.ProductName}
 }
}
if ($sections['신뢰성 기록'].data.Count -gt 500) {$sections['신뢰성 기록'].data=@($sections['신뢰성 기록'].data | Select-Object -First 500);$sections['신뢰성 기록'].status='일부 수집 / 500건 제한'}
Collect 'WER 보고서' {
 $roots=@((Join-Path $env:ProgramData 'Microsoft\Windows\WER\ReportArchive'),(Join-Path $env:ProgramData 'Microsoft\Windows\WER\ReportQueue'))
 foreach ($root in $roots) {
  if (!(Test-Path -LiteralPath $root)) {$issues.Add('WER 진단 폴더 없음 또는 접근 불가');continue}
  if ((Get-Item -LiteralPath $root).Attributes -band [IO.FileAttributes]::ReparsePoint) {$issues.Add('WER 재분석 지점 건너뜀');continue}
  $dirs=@(Get-ChildItem -LiteralPath $root -Directory | Where-Object {$_.Name -match '^(Kernel_|AppCrash_dwm|AppCrash_explorer)' -and $_.LastWriteTime -ge $from.LocalDateTime} | Select-Object -First 101)
  if ($dirs.Count -gt 100) {$issues.Add('WER 폴더별 100개 제한')}
  foreach ($dir in ($dirs | Select-Object -First 100)) {
   if ($dir.Attributes -band [IO.FileAttributes]::ReparsePoint) {continue}
   $file=Join-Path $dir.FullName 'Report.wer'
   if (Test-Path -LiteralPath $file) {
    $info=Get-Item -LiteralPath $file
    if ($info.Length -gt 262144 -or ($info.Attributes -band [IO.FileAttributes]::ReparsePoint)) {$issues.Add('WER 크기/재분석 지점 제한으로 건너뜀');continue}
    $safe=@(Get-Content -LiteralPath $file | Where-Object {$_ -match '^(EventType=(LiveKernelEvent|APPCRASH)|Sig\[0\]\.Value=[0-9a-fA-F]{1,8})$'})
    $stamp=([DateTimeOffset]$info.LastWriteTime).ToString('o')
    if ($safe -match '^EventType=LiveKernelEvent$') {
     $relevance='커널 WER 단서 / 파일 수정 시각, 사건 시각 미확정'
     if ($safe -match '^Sig\[0\]\.Value=(117|141)$') {$relevance='그래픽 WER 단서 / 파일 수정 시각, 사건 시각 미확정'}
     $events.Add([pscustomobject]@{time=$stamp;source='WER Report.wer';id='';severity='보고서';description=($safe -join '; ');relevance=$relevance})
    }
    [pscustomobject]@{Time=$stamp;Size=$info.Length;Fields=$safe}
   }
  }
 }
}
Collect '덤프 메타데이터' {
 foreach ($relative in @('LiveKernelReports','LiveKernelReports\WATCHDOG','LiveKernelReports\DISPLAY','Minidump')) {
  $root=Join-Path $env:SystemRoot $relative
  if (!(Test-Path -LiteralPath $root)) {$issues.Add($relative+': 폴더 없음 또는 접근 불가');continue}
  if ((Get-Item -LiteralPath $root).Attributes -band [IO.FileAttributes]::ReparsePoint) {$issues.Add($relative+': 재분석 지점 건너뜀');continue}
  $dumpFiles=@(Get-ChildItem -LiteralPath $root -File -Filter '*.dmp' | Where-Object {$_.LastWriteTime -ge $from.LocalDateTime} | Select-Object -First 101)
  if ($dumpFiles.Count -gt 100) {$issues.Add($relative+': 덤프 메타데이터 100개 제한')}
  $dumpFiles | Select-Object -First 100 | ForEach-Object {
   [pscustomobject]@{Location=$relative;Time=([DateTimeOffset]$_.LastWriteTime).ToString('o');Size=$_.Length}
  }
 }
}
$issues.Add('신뢰성 기록은 Windows/dwm/explorer 제품명에 한정됩니다. WER 파일과 덤프는 최근 30일의 지정 폴더만 조회합니다. 덤프 폴더당 최대 100개이며 전체 덤프 탐색·내용 분석은 하지 않습니다. WER 시간은 파일 수정 시각으로 실제 사건 시각과 다를 수 있습니다.')
$issues.Add('이벤트 원본 자유문은 개인정보 최소화를 위해 제외했습니다. 상세 하드웨어 판정에는 해당 시각의 이벤트 뷰어 원문을 현장에서 확인해야 할 수 있습니다. 로그 보존/삭제 이전 기록과 짧은 신호 단절은 확인할 수 없습니다.')
[pscustomobject]@{sections=$sections;events=@($events.ToArray());issues=@($issues.ToArray())} | ConvertTo-Json -Depth 10 -Compress
