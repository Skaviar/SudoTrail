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

test('technique guides point to the right sections, including pattern variants', () => {
  const expected = new Map([
    ['Hidden Single', 'tech_singles.php#h1'],
    ['Peer Elimination', 'tech_intro.php#te'],
    ['Locked Candidates — Claiming', 'tech_intersections.php#lc2'],
    ['Naked Pair', 'tech_naked.php#n2'],
    ['Hidden Quadruple', 'tech_hidden.php#h4'],
    ['XY-Wing', 'tech_wings.php#xy'],
    ['X-Wing', 'tech_fishb.php#bf2'],
    ['Leviathan', 'tech_fishb.php#bf5'],
    ['Sashimi Swordfish', 'tech_fishfs.php#fbf3'],
    ['Finned Whale', 'tech_fishfs.php#fbf567'],
    ['Finned Franken X-Wing', 'tech_fishc.php#ff'],
    ['Sashimi Mutant Jellyfish', 'tech_fishc.php#mf'],
    ['Siamese Finned Franken Swordfish', 'tech_fishc.php#sf'],
    ['Unique Rectangle Type 6', 'tech_ur.php#u6'],
    ['Unique Rectangle Type 4 (Missing Candidates)', 'tech_ur.php#umc'],
    ['Hidden Rectangle (Missing Candidates)', 'tech_ur.php#umc'],
    ['Avoidable Rectangle Type 2', 'tech_ur.php#ar'],
    ['Simple Coloring — Color Trap', 'tech_col.php#sc'],
    ['Alternating Inference Chain', 'tech_chains.php#nl'],
    ['Doubly Linked ALS-XZ', 'tech_als.php#axz'],
    ['ALS-XY-Wing', 'tech_als.php#axy'],
    ['Forcing Net', 'tech_last.php#fn']
  ]);
  for (const [name, path] of expected) assert.equal(training.guideUrl(name), 'https://hodoku.sourceforge.net/en/' + path, name);
  assert.equal(training.guideUrl('3D Medusa'), 'https://www.sudokuwiki.org/3d_medusa');
  for (const name of ['', undefined, null, 'constructor', 'Unknown technique']) assert.equal(training.guideUrl(name), null);
});

test('every technique advertised by the embedded practice bank has a guide', () => {
  const bank = JSON.parse(readFileSync(new URL('../internal/puzzlebank/puzzles.json', import.meta.url), 'utf8'));
  for (const name of new Set(bank.flatMap(puzzle => puzzle.techniques))) assert.ok(training.guideUrl(name), name);
});
