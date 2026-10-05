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
    <p>내장·외장 GPU와 드라이버를 확인하고, 화면 깜박임과 관련된 기록을 분석합니다.</p>
    <div class="button-row">
      <button class="secondary" id="scanGraphicsButton">그래픽 검사 / 업데이트 후 재검사</button>
      <button class="secondary" id="markFlickerButton">지금 깜박였어요</button>
    </div>
    <p class="graphics-status" id="graphicsStatus" role="status"></p>
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
  let busy = false;
  let report = null;
  const answers = () => ({ taskManager: el('graphicsTaskManager').value, scope: el('graphicsScope').value, recent: el('graphicsRecent').value });
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
      monitors.append(textNode('p', `${monitor.name} · ${monitor.adapter} · ${monitor.width || '?'} × ${monitor.height || '?'} · ${monitor.hz || '?'} Hz`, true));
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
    finally { busy = false; panel.querySelectorAll('button').forEach(button => { button.disabled = false; }); }
  }
  const scan = method => run(async () => { render(await call(method)); el('graphicsStatus').textContent = '검사를 마쳤습니다. 진단 근거와 수집 제한을 확인해 주세요.'; });
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
