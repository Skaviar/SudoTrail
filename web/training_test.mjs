import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

const window = {};
vm.runInNewContext(readFileSync(new URL('./training.js', import.meta.url), 'utf8'), { window });
const training = window.godokuTraining;

test('difficulty descriptions explain the progression from singles to chains', () => {
  assert.match(training.levels.easy.detail, /only/);
  assert.match(training.levels.medium.detail, /group/);
  assert.match(training.levels.hard.detail, /Wings/);
  assert.match(training.levels.expert.detail, /chains/);
  assert.match(training.describe('XY-Wing'), /pivot/);
  assert.match(training.describe('Hidden Pair'), /numbers confined/);
});

test('practice targets only the exact advertised technique after the initial state', () => {
  const states = [{ step: null }, { step: { technique: 'Peer Elimination' } }, { step: { technique: 'XYZ-Wing' } }, { step: { technique: 'XY-Wing' } }];
  assert.equal(training.targetIndex(states, 'XY-Wing'), 3);
  assert.equal(training.targetIndex(states, 'Naked Pair'), -1);
  assert.equal(training.targetIndex([{ step: { technique: 'XY-Wing' } }], 'XY-Wing'), -1);
});
