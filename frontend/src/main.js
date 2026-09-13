import './style.css';
import projectLicense from '../../LICENSE?raw';
import thirdPartyNotices from '../../THIRD_PARTY_NOTICES.txt?raw';
import {
  StartRepair,
  CancelRepair,
  CancelScheduledShutdown,
  IsRunning,
  OpenMicrosoftHelp,
  OpenQuickAssist,
  OpenWindowsUpdateTroubleshooter,
  OpenWindowsUpdate,
  ResetWindowsUpdate,
  EventsOn,
  GetAppVersion,
  CheckAppUpdate,
  DownloadAppUpdate,
  InstallAppUpdate,
  StartDefenderQuickScan,
  OpenDefenderSecurity,
} from './backend.js';

const app = document.querySelector('#app');

app.innerHTML = `
  <main class="shell">
    <header class="topbar">
      <div class="brand-mark" aria-hidden="true">
        <svg viewBox="0 0 24 24"><path d="M4 5.2 10.7 4v7.2H4V5.2Zm7.7-1.4L20 2.6v8.6h-8.3V3.8ZM4 12.2h6.7V20L4 18.8v-6.6Zm7.7 0H20v9.2l-8.3-1.2v-8Z"/></svg>
      </div>
      <div><p class="eyebrow">WINDOWS CARE</p><h1>시스템 복구 도우미 <span class="title-version">(앱 버전 <span id="appVersion">확인 중…</span>)</span></h1></div>
      <div class="header-actions"><span class="admin-badge"><span class="shield">◆</span> 관리자 권한</span><button class="developer-button" id="developerInfoButton">개발자 정보</button></div>
    </header>

    <section class="app-update" id="appUpdateNotice" aria-label="앱 업데이트" hidden>
      <p id="appUpdateMessage" role="status" aria-live="polite"></p>
      <button class="settings-button" id="downloadAppUpdate" hidden>업데이트 준비</button>
    </section>

    <section class="hero" id="hero">
      <div class="hero-copy">
        <span class="status-pill" id="statusPill"><i></i> 검사 준비됨</span>
        <h2 id="title">Windows가 평소 같지 않나요?</h2>
        <p id="message">버튼 한 번으로 손상되거나 누락된 Windows 시스템 파일을 검사하고 복구할 수 있습니다.</p>
        <p class="repair-restart-notice" id="repairRestartNotice" role="status" hidden>이전에 중단한 뒤 다시 시작한 경우, 5분이 지나도 진행되지 않으면 컴퓨터를 재시작한 후 다시 실행해 주세요. 50% 이상에서 멈춘 경우, 정상적인 복구 과정이니 기다려주세요.</p>
      </div>
      <div class="health-orb" id="healthOrb" aria-hidden="true">
        <div class="orb-ring"></div>
        <svg viewBox="0 0 32 32"><path d="M16 3 27 7v7.6c0 6.6-4.7 11.8-11 14.4C9.7 26.4 5 21.2 5 14.6V7l11-4Z"/><path class="check" d="m10.5 16 3.7 3.8 7.6-8"/></svg>
      </div>
    </section>

    <section class="progress-panel" id="progressPanel" hidden>
      <button class="danger-button" id="cancelRepairButton" hidden>검사 및 복구 취소</button>
      <div class="progress-heading"><span id="progressLabel">준비 중</span><strong id="progressValue">0%</strong></div>
      <div class="progress-track"><div id="progressBar"></div></div>
      <div class="steps">
        <div class="step" data-step="dism"><span>1</span><div><strong>Windows 구성 요소 복구</strong><small>DISM 상태 복원</small></div></div>
        <div class="step" data-step="sfc"><span>2</span><div><strong>시스템 파일 검사</strong><small>SFC 정밀 검사</small></div></div>
      </div>
      <details class="details"><summary>자세한 진행 내용 보기</summary><pre id="output" aria-live="polite"></pre></details>
    </section>

    <section class="action-panel">
      <div class="shutdown-options">
        <label><input type="checkbox" id="shutdownAfterRepair"> 검사 및 복구 완료 후 5분 뒤 컴퓨터 종료</label>
        <p class="hint">작업 중인 문서는 먼저 저장해 주세요. 복구 도우미를 닫아도 컴퓨터는 예약된 시간에 꺼집니다.</p>
        <button class="settings-button" id="cancelShutdownButton">컴퓨터 종료 예약 취소</button>
        <p id="shutdownStatus" role="status" aria-live="polite"></p>
      </div>
      <div class="button-row">
        <button class="primary" id="startButton">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 4.5 6v5.2c0 4.5 3.2 8.1 7.5 9.8 4.3-1.7 7.5-5.3 7.5-9.8V6L12 3Z"/><path d="m8.8 12 2.1 2.1 4.5-4.5"/></svg>
          <span>검사 및 복구 시작</span>
        </button>
        <button class="secondary" id="quickAssistButton">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 13v-2a7 7 0 0 1 14 0v2"/><path d="M5 13H3.8A1.8 1.8 0 0 0 2 14.8v2.4A1.8 1.8 0 0 0 3.8 19H6v-6H5Zm14 0h1.2a1.8 1.8 0 0 1 1.8 1.8v2.4a1.8 1.8 0 0 1-1.8 1.8H18v-6h1Z"/><path d="M18 19c-1 1.3-2.5 2-4.5 2"/></svg>
          <span>빠른 지원 열기</span>
        </button>
        <button class="secondary" id="updateTroubleshooterButton">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 12a8 8 0 1 1-2.35-5.65"/><path d="M20 4v5h-5"/><path d="M12 8v4l2.7 1.8"/></svg>
          <span>업데이트 문제 해결</span>
        </button>
        <button class="secondary" id="defenderQuickScanButton">
          <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 4.5 6v5.2c0 4.5 3.2 8.1 7.5 9.8 4.3-1.7 7.5-5.3 7.5-9.8V6L12 3Z"/><path d="m8.8 12 2.1 2.1 4.5-4.5"/></svg>
          <span>Defender 빠른 검사</span>
        </button>
      </div>
      <p class="hint">작업에는 10분 이상 걸릴 수 있습니다. 급히 나가야 한다면 검사 및 복구를 취소한 뒤 앱을 닫으세요. 나중에 다시 검사할 수 있습니다.</p>
      <p class="defender-status" id="defenderStatus" role="status" hidden></p>
      <button class="text-link" id="defenderSecurityButton">Windows 보안 · 검사 결과 보기 ↗</button>
    </section>

    <details class="advanced-panel">
      <summary>
        <span class="tool-icon">⚙</span>
        <span><strong>고급 복구 도구</strong><small>문제 해결사로 해결되지 않을 때만 사용하세요</small></span>
        <span class="chevron">⌄</span>
      </summary>
      <div class="advanced-content">
        <div class="warning-box">
          <strong>Windows 업데이트 캐시 초기화</strong>
          <p>손상됐을 수 있는 다운로드 캐시를 백업하고 새로 만듭니다. 설치된 업데이트와 개인 파일은 삭제하지 않지만 업데이트 파일을 다시 내려받아야 합니다.</p>
        </div>
        <button class="danger-button" id="resetUpdateButton">업데이트 캐시 초기화</button>
        <div class="update-progress" id="updateProgress" hidden>
          <div class="progress-heading"><span id="updateTitle">초기화 준비 중</span><strong id="updateValue">0%</strong></div>
          <div class="progress-track"><div id="updateBar"></div></div>
          <p id="updateMessage"></p>
          <pre id="updateOutput" aria-live="polite"></pre>
          <button class="settings-button" id="openUpdateButton" hidden>Windows 업데이트 열기</button>
        </div>
      </div>
    </details>

    <section class="explanation">
      <div class="info-icon">i</div>
      <div><h3>어떤 작업을 하나요?</h3><p>Microsoft 권장 순서에 따라 먼저 Windows 구성 요소 저장소를 복구한 다음, 보호된 시스템 파일을 검사합니다. 개인 파일과 설치된 앱은 삭제하지 않습니다.</p></div>
      <button class="text-link" id="helpButton">Microsoft 안내 보기 <span>↗</span></button>
    </section>
  </main>
  <dialog class="tool-overview developer-info" id="developerInfo" aria-labelledby="developerTitle">
    <form method="dialog">
      <h2 id="developerTitle">개발자 정보</h2>
      <p><strong>Ayin Kim</strong></p>
      <p>연락처: <a href="mailto:starmireu@gmail.com" id="developerEmail">starmireu@gmail.com</a></p>
      <p class="developer-purpose">Windows 내장 복구 기능을 실행하는 독립 도구입니다. Microsoft가 제작하거나 보증하는 앱이 아닙니다.</p>
      <p>MIT License · © 2026 Ayin Kim</p>
      <p class="developer-license-note">자체 코드에는 MIT 라이선스가 적용됩니다. 외부 구성 요소에는 각각의 라이선스가 적용됩니다.</p>
      <button type="button" class="settings-button" id="licensesButton">라이선스 및 외부 구성 요소 고지 보기</button>
      <div class="overview-actions"><button class="secondary" autofocus>닫기</button></div>
    </form>
  </dialog>
  <dialog class="tool-overview license-dialog" id="licensesDialog" aria-labelledby="licensesTitle">
    <form method="dialog">
      <h2 id="licensesTitle">라이선스 및 외부 구성 요소 고지</h2>
      <p>앱에 포함된 라이선스 원문을 인터넷 연결 없이 확인할 수 있습니다. Microsoft WebView2의 이용 조건은 Microsoft가 제공하는 약관을 확인해 주세요.</p>
      <p>화면 표시에는 Microsoft WebView2를 사용합니다. WebView2의 Microsoft Defender SmartScreen은 Microsoft 개인정보처리방침에 따라 이용 정보를 Microsoft에 전송할 수 있습니다. 아래 외부 구성 요소 고지에서 관련 안내와 약관을 확인할 수 있습니다.</p>
      <details open><summary>자체 코드 · MIT License</summary><pre id="projectLicenseText" class="license-text" tabindex="0" aria-label="자체 코드 라이선스 원문"></pre></details>
      <details><summary>외부 구성 요소 · Third-party notices</summary><pre id="thirdPartyLicenseText" class="license-text" tabindex="0" aria-label="외부 구성 요소 라이선스 원문"></pre></details>
      <div class="overview-actions"><button class="secondary" autofocus>닫기</button></div>
    </form>
  </dialog>
  <dialog class="tool-overview" id="toolOverview" aria-labelledby="overviewTitle" aria-describedby="overviewDescription">
    <form method="dialog">
      <p class="eyebrow">시작 전 안내</p>
      <h2 id="overviewTitle"></h2>
      <p id="overviewDescription"></p>
      <ul id="overviewPoints"></ul>
      <div class="overview-actions">
        <button class="secondary" value="cancel" autofocus>돌아가기</button>
        <button class="primary" value="start">시작하기</button>
      </div>
    </form>
  </dialog>`;

document.querySelector('#developerInfoButton').addEventListener('click', () => document.querySelector('#developerInfo').showModal());

document.querySelector('#projectLicenseText').textContent = projectLicense;
document.querySelector('#thirdPartyLicenseText').textContent = thirdPartyNotices;
document.querySelector('#licensesButton').addEventListener('click', () => {
  document.querySelector('#licensesDialog').showModal();
});

const toolOverviews = {
  installUpdate: {
    title: '새 버전을 설치하시겠습니까?',
    description: '업데이트 다운로드와 파일 검증이 완료되었습니다.',
    points: ['설치하면 현재 앱이 종료됩니다. 무설치 버전은 파일을 교체한 뒤 자동으로 다시 실행합니다.', '검사 또는 복구 중에는 설치할 수 없습니다. 작업을 마친 뒤 설치해 주세요.', '나중에 설치해도 현재 앱을 계속 사용할 수 있습니다.'],
    startLabel: '설치하기',
    cancelLabel: '나중에',
  },
  resetUpdate: {
    title: 'Windows 업데이트 캐시 초기화',
    description: '손상됐을 수 있는 다운로드 캐시를 백업하고 새로 만듭니다.',
    points: ['설치된 업데이트와 개인 파일은 삭제하지 않지만, 업데이트 파일을 다시 내려받아야 합니다.', '업데이트 관련 서비스를 잠시 중지하고, 작업 후 원래 상태로 복원합니다. 서비스 복원이 끝날 때까지 앱을 닫지 마세요.', '초기화가 끝나면 PC를 다시 시작한 뒤 Windows 업데이트를 확인해 주세요.'],
  },
  repair: {
    title: '시스템 검사 및 복구',
    description: 'Windows 구성 요소와 시스템 파일을 차례로 검사하고 손상된 파일을 복구합니다.',
    points: ['인터넷 연결이 필요할 수 있으며, 10분 이상 걸릴 수 있습니다.', '개인 파일과 설치된 앱은 삭제하지 않습니다.', '급히 나가야 한다면 ‘검사 및 복구 취소’를 누르고 취소 완료 후 앱을 닫으세요. 나중에 다시 검사할 수 있습니다.', '취소해도 이미 적용된 변경은 유지됩니다. 복구가 끝나지 않았을 수 있으므로 필요하면 재시작 후 다시 검사해 주세요.'],
  },
  assist: {
    title: '빠른 지원',
    description: 'Microsoft 빠른 지원 앱을 열어 도움을 주는 사람과 원격 지원을 시작합니다.',
    points: ['빠른 지원 앱과 인터넷 연결이 필요합니다.', '지원자가 알려준 보안 코드를 입력한 뒤 화면 공유를 승인합니다.', '신뢰하는 사람에게만 화면 공유와 제어를 허용해 주세요.'],
  },
  update: {
    title: 'Windows 업데이트 문제 해결',
    description: 'Microsoft 도움말 보기 앱에서 Windows 업데이트 문제 해결사를 엽니다.',
    points: ['업데이트 설치가 실패하거나 진행이 멈췄을 때 사용합니다.', '열린 창의 안내에 따라 진단과 해결을 진행해 주세요.', '인터넷 연결이 필요할 수 있습니다.'],
  },
  defender: {
    title: 'Defender 빠른 검사',
    description: 'Microsoft Defender로 위협이 자주 발견되는 주요 영역을 검사합니다.',
    points: ['Microsoft Defender가 활성화되어 있어야 합니다.', '전체 디스크 검사가 아니며, 검사 시간은 PC 상태에 따라 달라집니다.', '탐지 및 조치 결과는 Windows 보안에서 확인할 수 있습니다.'],
  },
};

function showToolOverview(name) {
  const dialog = document.querySelector('#toolOverview');
  if (dialog.open) return Promise.resolve(false);
  const overview = toolOverviews[name];
  document.querySelector('#overviewTitle').textContent = overview.title;
  document.querySelector('#overviewDescription').textContent = overview.description;
  document.querySelector('#overviewPoints').replaceChildren(...overview.points.map(point => {
    const item = document.createElement('li');
    item.textContent = point;
    return item;
  }));
  dialog.querySelector('[value="start"]').textContent = overview.startLabel || '시작하기';
  dialog.querySelector('[value="cancel"]').textContent = overview.cancelLabel || '돌아가기';
  dialog.returnValue = '';
  return new Promise(resolve => {
    dialog.addEventListener('close', () => resolve(dialog.returnValue === 'start'), { once: true });
    dialog.oncancel = () => { dialog.returnValue = 'cancel'; };
    dialog.showModal();
  });
}


const appUpdateMessage = document.querySelector('#appUpdateMessage');
const defenderButton = document.querySelector('#defenderQuickScanButton');
const defenderStatus = document.querySelector('#defenderStatus');
defenderButton.addEventListener('click', async () => {
  if (!await showToolOverview('defender')) return;
  defenderButton.disabled = true;
  defenderButton.querySelector('span').textContent = '빠른 검사 진행 중…';
  defenderStatus.hidden = false;
  defenderStatus.textContent = 'Defender 빠른 검사를 실행하고 있습니다. 완료될 때까지 기다려 주세요.';
  try {
    await StartDefenderQuickScan();
    defenderStatus.textContent = '빠른 검사 명령이 완료되었습니다. 위협 탐지 및 조치 결과는 Windows 보안에서 확인해 주세요.';
  } catch (error) {
    defenderStatus.textContent = String(error);
  } finally {
    defenderButton.disabled = false;
    defenderButton.querySelector('span').textContent = 'Defender 빠른 검사';
  }
});
document.querySelector('#defenderSecurityButton').addEventListener('click', async () => {
  try { await OpenDefenderSecurity(); }
  catch (error) { defenderStatus.hidden = false; defenderStatus.textContent = String(error); }
});
const downloadAppUpdateButton = document.querySelector('#downloadAppUpdate');
GetAppVersion().then(version => {
  document.querySelector('#appVersion').textContent = version;
}).catch(() => { document.querySelector('#appVersion').textContent = '확인 불가'; });

let appUpdateReady = false;
async function offerUpdateInstallation() {
  if (!await showToolOverview('installUpdate')) return;
  try {
    await InstallAppUpdate();
  } catch (error) {
    appUpdateMessage.textContent = String(error);
  }
}

async function prepareAppUpdate() {
  downloadAppUpdateButton.hidden = false;
  downloadAppUpdateButton.disabled = true;
  downloadAppUpdateButton.textContent = '다운로드 중…';
  appUpdateMessage.textContent = '새 버전을 자동으로 다운로드하고 있습니다. 앱은 계속 사용할 수 있습니다.';
  try {
    await DownloadAppUpdate();
    appUpdateReady = true;
    downloadAppUpdateButton.textContent = '업데이트 설치';
    appUpdateMessage.textContent = '새 버전이 준비되었습니다. 편할 때 설치해 주세요.';
    // Do not interrupt an operation or replace another feature's overview.
    if (!await IsRunning()) await offerUpdateInstallation();
  } catch (error) {
    appUpdateMessage.textContent = String(error);
    downloadAppUpdateButton.textContent = '다운로드 다시 시도';
  } finally {
    downloadAppUpdateButton.disabled = false;
  }
}

async function checkForUpdatesOnStartup() {
  try {
    const info = await CheckAppUpdate();
    if (info.available) {
      document.querySelector('#appUpdateNotice').hidden = false;
      appUpdateMessage.textContent = info.message;
      if (info.canDownload) await prepareAppUpdate();
    }
  } catch {
    document.querySelector('#appVersion').title = '이번 실행에서는 업데이트를 확인하지 못했습니다. 다음 실행 시 다시 확인합니다.';
  }
}
checkForUpdatesOnStartup();

downloadAppUpdateButton.addEventListener('click', async () => {
  if (appUpdateReady) await offerUpdateInstallation();
  else await prepareAppUpdate();
});

const els = {
  start: document.querySelector('#startButton'),
  help: document.querySelector('#helpButton'),
  quickAssist: document.querySelector('#quickAssistButton'),
  updateTroubleshooter: document.querySelector('#updateTroubleshooterButton'),
  resetUpdate: document.querySelector('#resetUpdateButton'),
  updateProgress: document.querySelector('#updateProgress'),
  updateTitle: document.querySelector('#updateTitle'),
  updateValue: document.querySelector('#updateValue'),
  updateBar: document.querySelector('#updateBar'),
  updateMessage: document.querySelector('#updateMessage'),
  updateOutput: document.querySelector('#updateOutput'),
  openUpdate: document.querySelector('#openUpdateButton'),
  panel: document.querySelector('#progressPanel'),
  restartNotice: document.querySelector('#repairRestartNotice'),
  title: document.querySelector('#title'),
  message: document.querySelector('#message'),
  pill: document.querySelector('#statusPill'),
  bar: document.querySelector('#progressBar'),
  value: document.querySelector('#progressValue'),
  label: document.querySelector('#progressLabel'),
  output: document.querySelector('#output'),
  orb: document.querySelector('#healthOrb'),
};

const shutdownAfterRepair = document.querySelector('#shutdownAfterRepair');
const shutdownStatus = document.querySelector('#shutdownStatus');
const cancelShutdownButton = document.querySelector('#cancelShutdownButton');
cancelShutdownButton.addEventListener('click', async () => {
  cancelShutdownButton.disabled = true;
  try {
    await CancelScheduledShutdown();
    shutdownAfterRepair.checked = false;
    shutdownStatus.textContent = '종료 예약과 이번 복구 후 자동 종료 설정을 취소했습니다.';
  } catch (error) {
    shutdownStatus.textContent = String(error);
  } finally {
    cancelShutdownButton.disabled = false;
  }
});

const cancelRepairButton = document.querySelector('#cancelRepairButton');
cancelRepairButton.addEventListener('click', async () => {
  if (!window.confirm('진행 중인 복구 명령을 중단합니다. 이미 적용된 변경은 되돌리지 않으며, 재시작이 필요할 수 있습니다. 취소하시겠습니까?')) return;
  cancelRepairButton.disabled = true;
  cancelRepairButton.textContent = '취소 중…';
  try { await CancelRepair(); }
  catch (error) { window.alert(String(error)); cancelRepairButton.disabled = false; cancelRepairButton.textContent = '검사 및 복구 취소'; }
});

function setState(event) {
  shutdownAfterRepair.disabled = !['complete', 'error', 'cancelled'].includes(event.stage);
  if (event.shutdownScheduled) {
    shutdownStatus.textContent = '복구가 완료되어 5분 뒤 컴퓨터가 종료됩니다. 취소하려면 종료 예약 취소를 누르세요.';
  }
  els.restartNotice.hidden = ['complete', 'error', 'cancelled'].includes(event.stage);
  if (['complete', 'error', 'cancelled'].includes(event.stage)) {
    cancelRepairButton.hidden = true;
  }

  const progress = Math.max(0, Math.min(100, event.progress ?? 0));
  els.panel.hidden = false;
  els.title.textContent = event.title;
  els.message.textContent = event.message;
  els.label.textContent = event.title;
  els.value.textContent = `${progress}%`;
  els.bar.style.width = `${progress}%`;

  document.querySelectorAll('.step').forEach(step => {
    const order = { dism: 1, sfc: 2, complete: 3 };
    const current = order[event.stage] || 0;
    const own = order[step.dataset.step];
    step.classList.toggle('active', own === current);
    step.classList.toggle('done', own < current);
  });

  if (event.output) {
    els.output.textContent += `${event.output}\n`;
    els.output.scrollTop = els.output.scrollHeight;
  }

  if (event.stage === 'cancelled') {
    els.pill.textContent = '복구 취소됨';
    els.pill.className = 'status-pill';
    els.orb.className = 'health-orb';
    els.start.disabled = false;
    els.start.querySelector('span').textContent = '다시 검사하기';
  } else if (event.stage === 'complete') {
    els.pill.innerHTML = '<i></i> 복구 완료';
    els.pill.className = 'status-pill complete';
    els.orb.classList.remove('working');
    els.orb.classList.add('complete');
    els.start.disabled = false;
    els.start.querySelector('span').textContent = '다시 검사하기';
  } else if (event.stage === 'error') {
    els.pill.innerHTML = '<i></i> 확인 필요';
    els.pill.className = 'status-pill error';
    els.orb.classList.remove('working');
    els.orb.classList.add('error');
    els.start.disabled = false;
    els.start.querySelector('span').textContent = '다시 시도';
  } else {
    els.pill.innerHTML = '<i></i> 복구 진행 중';
    els.pill.className = 'status-pill working';
    els.orb.className = 'health-orb working';
  }
}

EventsOn('repair:event', setState);
EventsOn('update:event', event => {
  const progress = Math.max(0, Math.min(100, event.progress ?? 0));
  els.updateProgress.hidden = false;
  els.updateTitle.textContent = event.title;
  els.updateValue.textContent = `${progress}%`;
  els.updateBar.style.width = `${progress}%`;
  els.updateMessage.textContent = event.message;
  if (event.output) {
    els.updateOutput.textContent += `${event.output}\n`;
    els.updateOutput.scrollTop = els.updateOutput.scrollHeight;
  }
  if (event.stage === 'complete' || event.stage === 'error') {
    els.resetUpdate.disabled = false;
    els.resetUpdate.textContent = event.stage === 'complete' ? '다시 초기화' : '다시 시도';
    els.openUpdate.hidden = event.stage !== 'complete';
    els.updateProgress.classList.toggle('has-error', event.stage === 'error');
  }
});

els.start.addEventListener('click', async () => {
  if (!await showToolOverview('repair')) return;
  const shutdown = shutdownAfterRepair.checked;
  if (shutdown && !window.confirm('검사와 복구가 정상적으로 끝나면 5분 뒤 컴퓨터가 꺼집니다. 저장하지 않은 작업 내용은 사라질 수 있으니 먼저 저장해 주세요. 완료 후 컴퓨터를 끌까요?')) return;
  cancelRepairButton.hidden = false;
  cancelRepairButton.disabled = false;
  cancelRepairButton.textContent = '검사 및 복구 취소';
  els.start.disabled = true;
  els.output.textContent = '';
  els.start.querySelector('span').textContent = '복구 진행 중…';
  setState({ stage: 'starting', title: '검사 및 복구 준비 중', message: '검사 및 복구를 시작하고 있습니다.', progress: 0 });
  try {
    await StartRepair(shutdown);
  } catch (error) {
    setState({ stage: 'error', title: '시작할 수 없습니다', message: String(error), progress: 0 });
  }
});

els.help.addEventListener('click', () => OpenMicrosoftHelp());

els.quickAssist.addEventListener('click', async () => {
  if (!await showToolOverview('assist')) return;
  const label = els.quickAssist.querySelector('span');
  els.quickAssist.disabled = true;
  label.textContent = '빠른 지원 여는 중…';
  try {
    await OpenQuickAssist();
  } catch (error) {
    window.alert(`빠른 지원을 열 수 없습니다. Microsoft Store에서 빠른 지원 앱이 설치되어 있는지 확인해 주세요.\n\n${error}`);
  } finally {
    els.quickAssist.disabled = false;
    label.textContent = '빠른 지원 열기';
  }
});

els.updateTroubleshooter.addEventListener('click', async () => {
  if (!await showToolOverview('update')) return;
  try {
    await OpenWindowsUpdateTroubleshooter();
  } catch (error) {
    window.alert(`Windows 업데이트 문제 해결사를 열 수 없습니다.\n\n${error}`);
  }
});

els.resetUpdate.addEventListener('click', async () => {
  if (!await showToolOverview('resetUpdate')) return;
  els.resetUpdate.disabled = true;
  els.resetUpdate.textContent = '초기화 진행 중…';
  els.updateOutput.textContent = '';
  els.openUpdate.hidden = true;
  els.updateProgress.classList.remove('has-error');
  try {
    await ResetWindowsUpdate();
  } catch (error) {
    els.resetUpdate.disabled = false;
    els.resetUpdate.textContent = '다시 시도';
    window.alert(String(error));
  }
});

els.openUpdate.addEventListener('click', () => OpenWindowsUpdate());

IsRunning().then(running => {
  if (running) {
    els.start.disabled = true;
    els.panel.hidden = false;
  }
});
