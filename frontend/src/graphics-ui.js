import { t } from './i18n.js';
import { graphicsAdvice } from './graphics-advice.js';

const api = () => window.go?.main?.App;
async function call(method, ...args) {
  if (!api()?.[method]) throw new Error(t('그래픽 진단은 Windows 데스크톱 앱에서 사용할 수 있습니다.'));
  return api()[method](...args);
}
const vendorNames = { intel: 'Intel', amd: 'AMD', nvidia: 'NVIDIA', unknown: 'Unknown' };

export function mountGraphics() {
  const panel = document.createElement('section');
  panel.className = 'graphics-panel';
  panel.id = 'graphicsPanel';
  panel.innerHTML = `
    <h2>그래픽·화면 깜박임 진단</h2>
    <details class="graphics-questions" open><summary>하드웨어 원인 조사 (읽기 전용)</summary>
      <p>최근 30일의 시스템 사양과 관련 오류를 조사합니다. 설정 변경·복구·업데이트·재시작 없이 진행하며 CCTV 녹화 파일은 접근하지 않습니다. 결과 3개 파일은 바탕화면의 새 하드웨어진단 폴더에 저장합니다.</p>
      <label>깜박임 발생 시각 (선택, 이 PC의 현지 시각)<input id="hardwareIncident" type="datetime-local"></label>
      <label>모니터별 배선<input id="hardwareWiring" maxlength="2000" placeholder="PC 출력 → 변환기 방향/전원 → 모니터 입력; 모르면 미확인"></label>
      <label>증상과 동작 상태<textarea id="hardwareSymptoms" maxlength="4000" placeholder="어느 화면/동시 또는 순차/신호 없음/지속 시간/PC·녹화 지속 여부/장기간 전원 차단 후 변화. 개인정보는 입력하지 마세요."></textarea></label>
      <div class="button-row"><button class="secondary" id="hardwareInvestigate">하드웨어 조사 / 바탕화면 저장</button><button class="secondary" id="hardwareSave">수집 결과 다시 저장</button><button class="secondary" id="hardwareOpen" disabled>결과 폴더 열기</button></div>
      <p id="hardwareStatus" role="status" data-i18n-skip></p>
    </details>
    <p>내장·외장 GPU와 드라이버를 확인하고, 화면 깜박임과 관련된 기록을 분석합니다.</p>
    <div class="button-row">
      <button class="secondary" id="scanGraphicsButton">그래픽 검사 / 업데이트 후 재검사</button>
      <button class="secondary" id="markFlickerButton">지금 깜박였어요</button>
      <button class="secondary" id="flickerDiagnosticsButton">깜박임 원인 종합 진단 / 텍스트 저장</button>
      <button class="secondary" id="startDisplayWatch">디스플레이 변경 감시 시작</button>
      <button class="secondary" id="stopDisplayWatch" disabled>감시 종료</button>
      <button class="secondary" id="saveFlickerButton" disabled>종합 진단 텍스트 다시 저장</button>
    </div>
    <p class="graphics-status" id="graphicsStatus" role="status"></p>
    <p id="displayWatchStatus" role="status"></p>
    <p class="hint">종합 진단은 지난 7일을 검사합니다. 감시를 종료한 뒤 텍스트를 다시 저장하면 변경 기록이 포함됩니다.</p>
    <p class="hint">감시는 1초 간격이며 최대 30분 또는 변경 500건까지 기록합니다. 짧은 변화는 놓칠 수 있고 활성 앱이 변경 원인이라는 뜻은 아닙니다. 다시 시작하면 이전 감시 기록이 대체됩니다.</p>
    <details class="graphics-questions"><summary>복구 이후 증상과 모니터 정보</summary>
      <label>복구 시점·방식<input id="flickerRecovery" maxlength="1000" placeholder="예: 10월 5일, 시스템 복원 또는 DISM/SFC"></label>
      <label>어느 화면이 깜박이나요?<select id="flickerSide"><option value="unknown">모르겠어요</option><option value="left">좌측</option><option value="right">우측</option><option value="both">양쪽</option></select></label>
      <label>양쪽 발생 시점<select id="flickerTiming"><option value="unknown">모르겠어요</option><option value="together">동시에</option><option value="separate">각각 불규칙하게</option></select></label>
      <label>사용 앱·추가 증상<textarea id="flickerNotes" maxlength="2000"></textarea></label>
      <p class="hint">모니터 좌표는 Windows 배치 기준입니다. 보고서에는 앱·경로·접속 주소 등 시스템 정보가 포함될 수 있습니다.</p>
    </details>
    <details id="flickerDetails" hidden><summary>CPU·보안 상태와 추가 수집 정보</summary><pre id="flickerRaw" data-i18n-skip></pre></details>
    <details id="displayWatchDetails" hidden><summary>디스플레이 변경 기록</summary><pre id="displayWatchRaw" data-i18n-skip></pre></details>
    <p class="hint">깜박임을 기록하면 전후 2분 범위를 확인합니다. 잠시 후 다시 검사하면 이후 기록도 포함합니다. 최근 기록이 없으면 지난 7일을 검사합니다.</p>
    <div id="graphicsResults" hidden>
      <p id="graphicsWindow" data-i18n-skip></p>
      <div id="graphicsDevices"></div>
      <h3>연결된 모니터</h3><div id="graphicsMonitors"></div>
      <h3>진단 근거와 다음 조치</h3><div id="graphicsFindings"></div>
      <p id="graphicsReboot" hidden>Windows에 재부팅 대기 상태가 있습니다. 그래픽 업데이트 때문인지는 확인되지 않았습니다.</p>
      <p id="graphicsChanges"></p>
      <details><summary>이벤트 원문과 수집 제한 보기</summary><pre id="graphicsRaw" data-i18n-skip></pre></details>
      <div class="button-row">
        <button class="secondary" id="saveGraphicsButton">그래픽 진단 보고서 저장</button>
        <button class="secondary" id="graphicsSettingsButton">고급 디스플레이 설정</button>
        <button class="secondary" id="graphicsDeviceManager">장치 관리자</button>
        <button class="secondary" id="graphicsOEM">PC 제조사 드라이버 지원</button>
      </div>
      <p class="hint">업데이트 도구에서 설치를 완료한 뒤 재검사하세요. 버전 변경은 확인할 수 있지만 깜박임 해결 여부는 직접 확인해야 합니다.</p>
    </div>
    <details class="graphics-questions"><summary>깜박임 증상으로 원인 좁히기</summary>
      <label>작업 관리자도 함께 깜박이나요?<select id="graphicsTaskManager"><option value="unknown">모르겠어요</option><option value="yes">네</option><option value="no">아니요</option></select></label>
      <label>어디에서 발생하나요?<select id="graphicsScope"><option value="unknown">모르겠어요</option><option value="app">특정 앱에서만</option><option value="external">외부 모니터에서만</option><option value="all">화면 전체에서</option></select></label>
      <label>드라이버 업데이트 직후 시작됐나요?<select id="graphicsRecent"><option value="unknown">모르겠어요</option><option value="yes">네</option><option value="no">아니요</option></select></label>
      <p id="graphicsAdvice"></p>
    </details>`;
  document.querySelector('.action-panel').after(panel);
  const el = id => panel.querySelector(`#${id}`);
  let hardwareSaved = false;
  let busy = false;
  let report = null;
  let comprehensiveAvailable = false;
  let watching = false;
  let watchTimer = null;
  let watchPollBusy = false;
  const syncButtons = () => {
    el('hardwareOpen').disabled = busy || !hardwareSaved;
    el('startDisplayWatch').disabled = busy || watching;
    el('stopDisplayWatch').disabled = busy || !watching;
    el('saveFlickerButton').disabled = busy || !comprehensiveAvailable;
  };
  function showWatch(value) {
    watching = value.running;
    el('displayWatchStatus').textContent = `${t(watching ? '감시 중' : '감시 종료')} · ${t('변경 기록')}: ${value.changes?.length || 0}`;
    el('displayWatchDetails').hidden = false;
    el('displayWatchRaw').textContent = JSON.stringify(value, null, 2);
    if (!watching && watchTimer) { clearInterval(watchTimer); watchTimer = null; }
    syncButtons();
  }
  async function pollWatch() {
    if (watchPollBusy || busy) return;
    watchPollBusy = true;
    try { const value = await call('GetDisplayWatch'); if (!busy) showWatch(value); }
    catch (error) { el('displayWatchStatus').textContent = t(String(error)); }
    finally { watchPollBusy = false; }
  }
  const answers = () => ({ taskManager: el('graphicsTaskManager').value, scope: el('graphicsScope').value, recent: el('graphicsRecent').value, recovery: el('flickerRecovery').value, side: el('flickerSide').value, timing: el('flickerTiming').value, notes: el('flickerNotes').value });
  const advise = () => { el('graphicsAdvice').textContent = graphicsAdvice(answers()).map(text => t(text)).join('\n'); };
  panel.querySelectorAll('select').forEach(select => select.addEventListener('change', advise));
  window.addEventListener('language:changed', advise);
  advise();
  function textNode(tag, text, raw = false) {
    const node = document.createElement(tag); node.textContent = text;
    if (raw) node.dataset.i18nSkip = '';
    return node;
  }
  function render(value) {
    report = value;
    el('flickerDetails').hidden = !value.comprehensive;
    el('flickerRaw').textContent = '';
    if (value.comprehensive) {
      el('flickerRaw').textContent = JSON.stringify(value.comprehensive, null, 2);
    }
    el('graphicsResults').hidden = false;
    el('graphicsWindow').textContent = `${value.manufacturer || ''} ${value.model || ''}\n${value.from} → ${value.until}`;
    const devices = el('graphicsDevices'); devices.replaceChildren();
    if (!value.gpus?.length) devices.append(textNode('p', 'GPU 정보를 가져오지 못했습니다. 수집 제한을 확인해 주세요.'));
    for (const gpu of value.gpus || []) {
      const card = document.createElement('article'); card.className = 'graphics-device';
      card.append(textNode('h3', gpu.name, true));
      card.append(textNode('p', `${vendorNames[gpu.vendor] || 'Unknown'} · ${gpu.version || '?'} · ${gpu.provider || '?'}`, true));
      card.append(textNode('p', '드라이버 날짜는 설치 날짜가 아닙니다.'));
      card.append(textNode('p', `${gpu.driverDate || '?'} · ${gpu.inf || '?'} · Device code: ${gpu.problemCode ?? '?'}`, true));
      devices.append(card);
    }
    for (const vendor of new Set((value.gpus || []).map(g => g.vendor).filter(v => ['intel','amd','nvidia'].includes(v)))) {
      const button = document.createElement('button'); button.className = 'secondary';
      button.textContent = `${vendorNames[vendor]} · ${t('업데이트 도구 실행')}`;
      button.addEventListener('click', () => run(async () => {
        const result = await call('StartGraphicsUpdate', vendor);
        el('graphicsStatus').textContent = result.message;
      })); devices.append(button);
    }
    const monitors = el('graphicsMonitors'); monitors.replaceChildren();
    if (!value.monitors?.length) monitors.append(textNode('p', '모니터 연결 정보를 확인하지 못했습니다.'));
    for (const monitor of value.monitors || []) {
      monitors.append(textNode('p', `${monitor.name} · ${monitor.adapter} · ${monitor.width || '?'} × ${monitor.height || '?'} · ${monitor.hz || '?'} Hz · ${monitor.device || ''} · (${monitor.modeKnown ? monitor.x : '?'}, ${monitor.modeKnown ? monitor.y : '?'})${monitor.primary ? ' · '+t('주 모니터') : ''}`, true));
    }
    const findings = el('graphicsFindings'); findings.replaceChildren();
    for (const finding of value.findings || []) {
      const item = document.createElement('article'); item.className = 'graphics-finding';
      item.append(textNode('strong',finding.title),textNode('p',finding.evidence,true),textNode('p',finding.action)); findings.append(item);
    }
    el('graphicsReboot').hidden = !value.restartPending;
    el('graphicsChanges').replaceChildren();
    if (value.baselineTime) {
      el('graphicsChanges').append(textNode('span','업데이트 전 저장 상태와 비교: '),textNode('span',value.baselineTime,true));
      if (value.changes?.length) for (const change of value.changes) el('graphicsChanges').append(textNode('span',`\n${change}`,true));
      else el('graphicsChanges').append(textNode('span',' · 드라이버 버전 변경이 확인되지 않았습니다.'));
    }
    el('graphicsRaw').textContent = JSON.stringify({ issues:value.issues, events:value.events },null,2);
  }
  async function run(action) {
    if (busy) return;
    busy = true;
    panel.querySelectorAll('button').forEach(button => { button.disabled = true; });
    el('graphicsStatus').textContent = '처리 중입니다…';
    try { await action(); }
    catch (error) { el('graphicsStatus').textContent = t(String(error)); }
    finally { busy = false; panel.querySelectorAll('button').forEach(button => { button.disabled = false; }); syncButtons(); }
  }
  const scan = method => run(async () => { render(await call(method)); el('graphicsStatus').textContent = '검사를 마쳤습니다. 진단 근거와 수집 제한을 확인해 주세요.'; });
  const showHardware = result => {
    hardwareSaved = true;
    el('hardwareStatus').textContent = `${result.summary}\n저장 위치: ${result.folder}`;
    el('graphicsStatus').textContent = '하드웨어 조사 결과를 저장했습니다.';
  };
  el('hardwareInvestigate').addEventListener('click', () => run(async () => {
    el('hardwareStatus').textContent = '읽기 전용 조사 중입니다. 최대 3분 정도 걸릴 수 있습니다.';
    try {
      const value = el('hardwareIncident').value;
      showHardware(await call('InvestigateHardware', { incident: value ? new Date(value).toISOString() : '', wiring: el('hardwareWiring').value, symptoms: el('hardwareSymptoms').value }));
    } catch (error) { el('hardwareStatus').textContent = `조사 또는 저장 실패: ${String(error)}. 수집이 완료됐다면 다시 저장할 수 있습니다.`; throw error; }
  }));
  el('hardwareSave').addEventListener('click', () => run(async () => showHardware(await call('SaveHardwareReport'))));
  el('hardwareOpen').addEventListener('click', () => run(async () => { await call('OpenHardwareFolder'); el('graphicsStatus').textContent = '결과 폴더를 열었습니다.'; }));
  el('flickerDiagnosticsButton').addEventListener('click', () => run(async () => {
    render(await call('ScanFlickerDiagnostics'));
    comprehensiveAvailable = true;
    const path = await call('ExportFlickerReport', JSON.stringify(answers()));
    el('graphicsStatus').textContent = path || t('저장을 취소했습니다.');
  }));
  el('saveFlickerButton').addEventListener('click', () => run(async () => {
    const path = await call('ExportFlickerReport', JSON.stringify(answers()));
    el('graphicsStatus').textContent = path || t('저장을 취소했습니다.');
  }));
  el('startDisplayWatch').addEventListener('click', () => run(async () => {
    showWatch(await call('StartDisplayWatch'));
    if (watching && !watchTimer) watchTimer = setInterval(pollWatch, 1000);
    el('graphicsStatus').textContent = '감시 중';
  }));
  el('stopDisplayWatch').addEventListener('click', () => run(async () => {
    showWatch(await call('StopDisplayWatch'));
    el('graphicsStatus').textContent = '감시 종료';
  }));
  window.addEventListener('pagehide', () => { if (watchTimer) clearInterval(watchTimer); });
  el('scanGraphicsButton').addEventListener('click', () => scan('ScanGraphics'));
  el('markFlickerButton').addEventListener('click', () => scan('MarkGraphicsFlicker'));
  el('saveGraphicsButton').addEventListener('click', () => run(async () => {
    if (!report) return;
    const path = await call('ExportGraphicsReport', JSON.stringify(answers()));
    el('graphicsStatus').textContent = path || t('저장을 취소했습니다.');
  }));
  for (const [id,method] of [['graphicsSettingsButton','OpenGraphicsSettings'],['graphicsDeviceManager','OpenDeviceManager'],['graphicsOEM','OpenPCDriverSupport']]) {
    el(id).addEventListener('click', () => run(async () => { const message = await call(method); el('graphicsStatus').textContent = message || t('Windows 도구를 열었습니다.'); }));
  }
}
