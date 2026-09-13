import test from 'node:test';
import assert from 'node:assert/strict';
import { guideNodes, resolveGuide } from '../src/guidance.js';

const routes = [
  ['first update attempt', ['update', 'online', 'new'], 'update', 'updateTroubleshooterButton'],
  ['failed basic update steps', ['update', 'online', 'basic'], 'cache', 'resetUpdateButton'],
  ['cache did not help', ['update', 'online', 'cache'], 'updateRepair', 'startButton'],
  ['repair did not help updates', ['update', 'online', 'all'], 'repeated', 'quickAssistButton'],
  ['offline update', ['update', 'offline'], 'network', null],
  ['unknown connectivity', ['update', 'unknown'], 'network', null],
  ['Windows-wide errors', ['system', 'windows', 'new'], 'repair', 'startButton'],
  ['repeat failure', ['system', 'windows', 'done'], 'repeated', 'quickAssistButton'],
  ['one application', ['system', 'oneapp'], 'singleApp', 'quickAssistButton'],
  ['hardware warning', ['system', 'hardware'], 'hardware', null],
  ['first security check', ['security', 'new'], 'security', 'defenderQuickScanButton'],
  ['security detection', ['security', 'found'], 'securityResults', 'defenderSecurityButton'],
  ['persistent security symptom', ['security', 'still'], 'securityResults', 'defenderSecurityButton'],
  ['cannot reach desktop', ['boot', 'no'], 'recovery', null],
  ['intermittent startup', ['boot', 'desktop', 'new'], 'repair', 'startButton'],
  ['slow without other errors', ['unsure', 'slow'], 'unsure', 'quickAssistButton'],
];
for (const [name, answers, id, target] of routes) {
  test(name, () => {
    const result = resolveGuide(answers);
    assert.equal(result.type, 'result');
    assert.equal(result.id, id);
    assert.equal(result.target, target);
    assert.equal(result.trail.length, answers.length);
  });
}

test('changing an earlier answer does not retain the previous recommendation', () => {
  assert.equal(resolveGuide(['update', 'online', 'basic']).id, 'cache');
  assert.equal(resolveGuide(['update', 'offline']).id, 'network');
  assert.equal(resolveGuide(['update']).type, 'question');
  assert.equal(resolveGuide([]).id, 'symptom');
});

test('unknown answers cannot bypass prerequisites for a cache reset', () => {
  assert.equal(resolveGuide(['update', 'basic']).id, 'updateConnection');
  assert.equal(resolveGuide(['<script>']).id, 'symptom');
  assert.equal(resolveGuide(['update', 'offline', 'basic']).id, 'network');
});

test('every choice reaches useful guidance within three questions and has no cycles', () => {
  function walk(id, path = []) {
    assert.ok(guideNodes[id], `missing node ${id}`);
    assert.ok(!path.includes(id), `cycle at ${id}`);
    const node = guideNodes[id];
    if (node.type === 'result') {
      assert.ok(path.length <= 3, `too many questions for ${id}`);
      for (const field of ['title', 'reason', 'action', 'after', 'limit']) assert.ok(node[field]);
      return;
    }
    assert.ok(node.choices.length >= 2);
    assert.equal(new Set(node.choices.map(c => c.value)).size, node.choices.length);
    for (const option of node.choices) walk(option.next, [...path, id]);
  }
  walk('symptom');
});
