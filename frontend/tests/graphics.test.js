import test from 'node:test';
import assert from 'node:assert/strict';
import { graphicsAdvice } from '../src/graphics-advice.js';

test('unknown symptoms do not accuse the Intel GPU', () => {
  const advice = graphicsAdvice({taskManager:'unknown',scope:'unknown',recent:'unknown'});
  assert.equal(advice.length,2);
  assert.match(advice.at(-1), /확정하지 않습니다/);
});
test('external monitor and update timing generate separate next steps', () => {
  const advice = graphicsAdvice({taskManager:'yes',scope:'external',recent:'yes'});
  assert.ok(advice.some(text => text.includes('케이블')));
  assert.ok(advice.some(text => text.includes('이전 드라이버')));
  assert.ok(advice.some(text => text.includes('디스플레이 드라이버')));
});
