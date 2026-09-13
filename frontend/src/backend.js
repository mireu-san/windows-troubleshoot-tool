const inWails = () => Boolean(window.go?.main?.App && window.runtime);

export function StartDefenderQuickScan() {
  if (inWails()) return window.go.main.App.StartDefenderQuickScan();
  return Promise.reject(new Error('Defender 빠른 검사는 Windows 데스크톱 앱에서 사용할 수 있습니다.'));
}

export function OpenDefenderSecurity() {
  if (inWails()) return window.go.main.App.OpenDefenderSecurity();
  return Promise.reject(new Error('Windows 보안은 Windows 데스크톱 앱에서 열어 주세요.'));
}

export function GetAppVersion() {
  return inWails() ? window.go.main.App.GetAppVersion() : Promise.resolve('미리보기');
}

export function CheckAppUpdate() {
  if (inWails()) return window.go.main.App.CheckAppUpdate();
  return Promise.reject(new Error('앱 업데이트 확인은 데스크톱 앱에서 사용할 수 있습니다.'));
}

export function DownloadAppUpdate() {
  if (inWails()) return window.go.main.App.DownloadAppUpdate();
  return Promise.reject(new Error('다운로드는 데스크톱 앱에서 사용할 수 있습니다.'));
}

export function OpenAppReleases() {
  if (inWails()) return window.go.main.App.OpenAppReleases();
  window.open('https://github.com/mireu-san/windows-troubleshoot-tool/releases', '_blank', 'noopener,noreferrer');
}

export function StartRepair(shutdown = false) {
  if (inWails()) return window.go.main.App.StartRepairWithShutdown(shutdown);
  return runDemo();
}

export function IsRunning() {
  if (inWails()) return window.go.main.App.IsRunning();
  return Promise.resolve(false);
}

export function OpenMicrosoftHelp() {
  if (inWails()) return window.go.main.App.OpenMicrosoftHelp();
  window.open(
    'https://support.microsoft.com/ko-kr/windows/experience/backup-recovery/use-the-system-file-checker-tool-to-repair-missing-or-corrupted-system-files',
    '_blank',
    'noopener,noreferrer',
  );
}

export function OpenQuickAssist() {
  if (inWails()) return window.go.main.App.OpenQuickAssist();
  window.open(
    'https://apps.microsoft.com/detail/9p7bp5vnwkx5',
    '_blank',
    'noopener,noreferrer',
  );
}

export function OpenWindowsUpdateTroubleshooter() {
  if (inWails()) return window.go.main.App.OpenWindowsUpdateTroubleshooter();
  window.open(
    'https://support.microsoft.com/ko-kr/windows/windows-update-%EB%AC%B8%EC%A0%9C-%ED%95%B4%EA%B2%B0%EC%82%AC-19bc41ca-ad72-ae67-af3c-89ce169755dd',
    '_blank',
    'noopener,noreferrer',
  );
}

export function OpenWindowsUpdate() {
  if (inWails()) return window.go.main.App.OpenWindowsUpdate();
  window.open('ms-settings:windowsupdate');
}

export function ResetWindowsUpdate() {
  if (inWails()) return window.go.main.App.ResetWindowsUpdate();
  return runUpdateDemo();
}

export function EventsOn(name, callback) {
  if (inWails()) return window.runtime.EventsOn(name, callback);
  window.addEventListener(name, event => callback(event.detail));
}

function emitDemo(detail) {
  window.dispatchEvent(new CustomEvent('repair:event', { detail }));
}

async function runDemo() {
  const updates = [
    ['dism', 8, 'Windows 구성 요소 복구 중', '이미지 버전 확인 중…'],
    ['dism', 31, 'Windows 구성 요소 복구 중', '구성 요소 저장소를 검사하고 있습니다…'],
    ['dism', 52, 'Windows 구성 요소 복구 중', '복원 작업을 완료했습니다.'],
    ['sfc', 58, '시스템 파일 검사 중', '시스템 검사를 시작합니다…'],
    ['sfc', 76, '시스템 파일 검사 중', '검증 50% 완료…'],
    ['sfc', 98, '시스템 파일 검사 중', '검증 100% 완료…'],
  ];
  for (const [stage, progress, title, output] of updates) {
    await new Promise(resolve => setTimeout(resolve, 650));
    emitDemo({ stage, progress, title, output, message: stage === 'dism' ? 'Windows 구성 요소를 복구하고 있습니다.' : '보호된 시스템 파일을 검사하고 있습니다.' });
  }
  emitDemo({ stage: 'complete', progress: 100, title: '검사와 복구가 완료되었습니다', message: '브라우저 미리보기용 데모가 완료되었습니다.' });
}

async function runUpdateDemo() {
  const updates = [
    ['stopping', 18, '업데이트 서비스 중지 중', 'BITS 서비스를 중지했습니다.'],
    ['resetting', 55, '업데이트 캐시 백업 중', 'SoftwareDistribution 폴더를 백업했습니다.'],
    ['starting', 82, '업데이트 서비스 다시 시작 중', '서비스를 다시 시작합니다.'],
  ];
  for (const [stage, progress, title, output] of updates) {
    await new Promise(resolve => setTimeout(resolve, 600));
    window.dispatchEvent(new CustomEvent('update:event', { detail: { stage, progress, title, output, message: 'Windows 업데이트 구성 요소를 안전하게 초기화하고 있습니다.' } }));
  }
  window.dispatchEvent(new CustomEvent('update:event', { detail: { stage: 'complete', progress: 100, title: '업데이트 캐시 초기화 완료', message: 'PC를 다시 시작한 다음 업데이트를 다시 확인하세요.' } }));
}

export function CancelRepair() {
  if (inWails()) return window.go.main.App.CancelRepair();
  return Promise.reject(new Error('취소는 Windows 데스크톱 앱에서 사용할 수 있습니다.'));
}

export function InstallAppUpdate() {
  if (inWails()) return window.go.main.App.InstallAppUpdate();
  return Promise.reject(new Error("업데이트 설치는 Windows 앱에서 사용할 수 있습니다."));
}

export function CancelScheduledShutdown() {
  if (inWails()) return window.go.main.App.CancelScheduledShutdown();
  return Promise.reject(new Error('종료 예약 취소는 Windows 데스크톱 앱에서 사용할 수 있습니다.'));
}

export function OpenRecoveryHelp() {
  if (inWails()) return window.go.main.App.OpenRecoveryHelp();
  window.open('https://support.microsoft.com/en-us/windows/experience/startup-boot/startup-repair', '_blank', 'noopener,noreferrer');
  return Promise.resolve();
}
