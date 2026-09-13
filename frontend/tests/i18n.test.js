import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { languageFor, resolveLanguage, createTranslator } from '../src/localization.js';
const translateText = createTranslator(JSON.parse(readFileSync(new URL('../src/locales/messages.json', import.meta.url), 'utf8')));
import { guideNodes } from '../src/guidance.js';

test('system language and explicit preferences', () => {
  for (const [locale, expected] of [['ko-KR','ko'], ['ja-JP','ja'], ['en-GB','en'], ['fr-FR','en'], ['KO_kr','ko'], ['', 'en']]) {
    assert.equal(languageFor(locale), expected);
    assert.equal(resolveLanguage('auto', locale), expected);
  }
  assert.equal(resolveLanguage('ja', 'ko-KR'), 'ja');
  assert.equal(resolveLanguage('invalid', 'ja-JP'), 'ja');
});
test('all questionnaire copy is available in English and Japanese', () => {
  for (const language of ['en', 'ja']) {
    for (const node of Object.values(guideNodes)) {
      const texts = ['title','description','reason','action','after','limit','tool'].map(key => node[key]).filter(Boolean);
      texts.push(...(node.choices || []).map(choice => choice.label));
      for (const text of texts) assert.doesNotMatch(translateText(text, language), /[가-힣]/, text);
    }
  }
});
test('formatted backend status and composed guidance retain values', () => {
  assert.equal(translateText('전체 작업 시간은 약 2분 3초입니다. 문제가 계속되면 Windows를 다시 시작해 주세요.', 'en'), 'Total time: about 2 min 3 sec. Restart Windows if the problem continues.');
  assert.equal(translateText('새 버전 2026.09.16을 사용할 수 있습니다.', 'en'), 'New version 2026.09.16 is available.');
  assert.equal(translateText('「검사 및 복구 시작」 버튼으로 이동', 'en'), 'Go to Start scan and repair');
  assert.match(translateText('DISM 단계에서 오류가 발생했습니다. 앱을 관리자 권한으로 실행했는지와 인터넷 연결을 확인한 뒤 다시 시도해 주세요.', 'ja'), /DISM/);
  assert.doesNotMatch(translateText('DISM 단계에서 오류가 발생했습니다. 앱을 관리자 권한으로 실행했는지와 인터넷 연결을 확인한 뒤 다시 시도해 주세요.', 'ja'), /[가-힣]/);
});
test('unknown Windows messages and Korean source remain unchanged', () => {
  assert.equal(translateText('Access denied: 0x80070005', 'ja'), 'Access denied: 0x80070005');
  assert.equal(translateText('검사 및 복구 시작', 'ko'), '검사 및 복구 시작');
});
