import { mountLanguagePicker } from './i18n.js';
import { resolveGuide } from './guidance.js';

export function mountGuidance({ isRunning, openRecoveryHelp }) {
  const banner = document.createElement('section');
  banner.className = 'guidance-banner';
  banner.setAttribute('aria-label', '내 상황에 맞는 도구 찾기');
  banner.innerHTML = `
    <div><h2>어떤 버튼을 눌러야 할지 모르겠나요?</h2><p>간단한 질문에 답하면, 먼저 해볼 도구와 그 이유를 알려드려요.</p></div>
    <button type="button" class="secondary" id="openGuidance">내 상황에 맞는 도구 찾기</button>
    <div class="guidance-selection" id="guidanceSelection" role="status" hidden></div>`;
  document.querySelector('#hero').after(banner);
  const dialog = document.createElement('dialog');
  dialog.id = 'guidanceDialog';
  dialog.className = 'tool-overview guidance-dialog';
  dialog.setAttribute('aria-labelledby', 'guidanceTitle');
  dialog.setAttribute('aria-describedby', 'guidanceDescription');
  dialog.innerHTML = `
    <div class="guidance-top"><span id="guidanceStep"></span><button type="button" class="text-link" id="closeGuidance">닫기</button></div>
    <h2 id="guidanceTitle" tabindex="-1"></h2>
    <p id="guidanceDescription"></p>
    <div id="guidanceBody"></div>
    <p id="guidanceFeedback" role="status"></p>
    <div class="overview-actions"><button type="button" class="secondary" id="guidanceBack">이전 질문</button><button type="button" class="primary" id="guidanceNext" hidden></button></div>
    <p class="guidance-footnote">선택한 답변은 서버에 보내거나 저장하지 않습니다. 증상에 맞는 사용 안내이며, 고장 원인을 확정하는 진단은 아닙니다.</p>`;
  document.body.append(dialog);
  mountLanguagePicker(dialog.querySelector('.guidance-top'));
  const el = id => dialog.querySelector(`#${id}`);
  let answers = [];
  let highlighted = null;
  let choosing = false;
  let revision = 0;
  const clearHighlight = () => {
    highlighted?.classList.remove('recommended-tool');
    highlighted?.removeAttribute('aria-describedby');
    highlighted = null;
  };
  const feedback = message => { el('guidanceFeedback').textContent = message; };
  function render() {
    revision++;
    const node = resolveGuide(answers);
    choosing = false;
    feedback('');
    el('guidanceNext').disabled = false;
    el('guidanceTitle').textContent = node.title;
    el('guidanceStep').textContent = node.type === 'question' ? '질문 {count} · 최대 3개'.replace('{count}', answers.length + 1) : '답변에 따른 안내';
    el('guidanceDescription').textContent = node.type === 'question' ? node.description : '추천 내용을 읽고, 원하시면 해당 도구로 이동하세요. 아직 어떤 작업도 시작하지 않았습니다.';
    el('guidanceBack').hidden = answers.length === 0;
    el('guidanceNext').hidden = node.type === 'question';
    const body = el('guidanceBody');
    body.replaceChildren();
    if (node.type === 'question') {
      const choices = document.createElement('div');
      choices.className = 'guidance-choices';
      choices.setAttribute('role', 'group');
      choices.setAttribute('aria-labelledby', 'guidanceTitle');
      for (const option of node.choices) {
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'guidance-choice';
        button.textContent = option.label;
        button.dataset.answer = option.value;
        button.addEventListener('click', () => { answers.push(option.value); render(); });
        choices.append(button);
      }
      body.append(choices);
    } else {
      const summary = document.createElement('p');
      summary.className = 'guidance-answers';
      summary.textContent = '선택한 상황: ' + node.trail.map(item => item.answer).join(' → ');
      body.append(summary);
      for (const [title, text] of [['왜 이 안내인가요?', node.reason], ['무엇을 하나요?', node.action], ['끝나면 무엇을 확인하나요?', node.after], ['알아두세요', node.limit]]) {
        const section = document.createElement('section');
        section.className = 'guidance-result-section';
        const heading = document.createElement('h3');
        heading.textContent = title;
        const paragraph = document.createElement('p');
        paragraph.textContent = text;
        section.append(heading, paragraph);
        body.append(section);
      }
      el('guidanceNext').textContent = node.target ? `「${node.tool}」 버튼으로 이동` : node.help ? 'Microsoft 부팅 복구 안내 보기' : '안내 확인했어요';
    }
    dialog.scrollTop = 0;
    el('guidanceTitle').focus({ preventScroll: true });
  }
  banner.querySelector('#openGuidance').addEventListener('click', () => {
    if (document.querySelector('dialog[open]')) return;
    answers = [];
    dialog.showModal();
    render();
  });
  el('closeGuidance').addEventListener('click', () => dialog.close());
  el('guidanceBack').addEventListener('click', () => { answers.pop(); render(); });
  dialog.addEventListener('close', () => { answers = []; revision++; });
  el('guidanceNext').addEventListener('click', async () => {
    if (choosing) return;
    const node = resolveGuide(answers);
    const currentRevision = revision;
    if (node.type !== 'result') return;
    choosing = true;
    el('guidanceNext').disabled = true;
    try {
      if (node.help) {
        await openRecoveryHelp();
        if (dialog.open) feedback('Microsoft 안내 페이지를 열었습니다. 문제가 생긴 PC에서 안내를 따라 주세요.');
        return;
      }
      const target = node.target ? document.getElementById(node.target) : null;
      if (node.target && !target) throw new Error('추천한 도구를 찾을 수 없습니다.');
      if (target && (await isRunning() || target.disabled)) {
        if (dialog.open) feedback('다른 검사나 복구가 진행 중입니다. 작업이 끝난 뒤 이 버튼으로 이동해 주세요.');
        return;
      }
      // An asynchronous status check must not act on a closed/restarted questionnaire.
      if (!dialog.open || revision !== currentRevision) return;
      clearHighlight();
      const selection = banner.querySelector('#guidanceSelection');
      selection.textContent = `${node.title}. ${node.after}${target ? ' 강조된 버튼을 누르면 실행 전 안내를 볼 수 있습니다.' : ''}`;
      selection.hidden = false;
      dialog.close();
      if (target) {
        for (let parent = target.parentElement; parent; parent = parent.parentElement) {
          if (parent.tagName === 'DETAILS') parent.open = true;
        }
        target.classList.add('recommended-tool');
        target.setAttribute('aria-describedby', 'guidanceSelection');
        highlighted = target;
        target.scrollIntoView({ block: 'center', behavior: 'auto' });
        target.focus({ preventScroll: true });
      }
    } catch (error) {
      if (dialog.open) feedback('안내를 열지 못했습니다. 잠시 후 다시 시도해 주세요. ' + String(error));
    } finally {
      choosing = false;
      el('guidanceNext').disabled = false;
    }
  });
}
