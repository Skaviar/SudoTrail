import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('./session.js', import.meta.url), 'utf8');
const key = 'godoku.session.v1';
const snapshot = () => ({
  original: [5, 3, 4, ...Array(78).fill(0)],
  givenClues: [5, 3, ...Array(79).fill(0)],
  transcribing: false, referencePhoto: '', puzzleLabel: 'Puzzle #42',
  reviewing: true, current: 7, candidates: false, guides: true
});

function browser(storage = new Map(), { blocked = false, quota = Infinity } = {}) {
  let writes = 0;
  const window = {};
  vm.runInNewContext(source, {
    window,
    localStorage: {
      getItem(name) { if (blocked) throw Error('Blocked'); return storage.get(name) ?? null; },
      setItem(name, text) {
        if (blocked || text.length > quota) throw Error('Storage unavailable');
        storage.set(name, text); writes++;
      },
      removeItem(name) { if (blocked) throw Error('Blocked'); storage.delete(name); }
    }
  });
  return { session: window.godokuSession, storage, writes: () => writes };
}

test('a new visit recovers edits, original clues, settings and replay position', () => {
  const first = browser();
  assert.equal(first.session.save(snapshot()), 'saved');
  assert.equal(first.session.save(snapshot()), 'saved');
  assert.equal(first.writes(), 1, 'unchanged data should not be written repeatedly');
  const returned = browser(first.storage).session.load();
  assert.deepEqual(JSON.parse(JSON.stringify(returned)), snapshot());
});

test('Clear removes the puzzle for subsequent visits', () => {
  const first = browser();
  first.session.save(snapshot());
  assert.equal(first.session.save(null), 'saved');
  assert.equal(browser(first.storage).session.load(), null);
  assert.equal(first.storage.has(key), false);
});

test('malformed or incompatible saves do not break startup', () => {
  for (const value of [
    '{broken', 'null',
    JSON.stringify({ version: 2, ...snapshot() }),
    JSON.stringify({ version: 1, ...snapshot(), original: [1, 2] }),
    JSON.stringify({ version: 1, ...snapshot(), givenClues: Array(81).fill(10) }),
    JSON.stringify({ version: 1, ...snapshot(), transcribing: 'yes' })
  ]) {
    assert.equal(browser(new Map([[key, value]])).session.load(), null);
  }
});

test('saved photo references stay local and invalid replay indices reset', () => {
  const stored = { version: 1, ...snapshot(), referencePhoto: 'https://example.com/image.jpg', current: -4 };
  const restored = browser(new Map([[key, JSON.stringify(stored)]])).session.load();
  assert.equal(restored.referencePhoto, '');
  assert.equal(restored.current, 0);
  assert.equal(restored.original[2], 4);
});

test('storage quota keeps the puzzle even when a photo does not fit', () => {
  const page = browser(new Map(), { quota: 1200 });
  const photo = { ...snapshot(), referencePhoto: 'data:image/jpeg;base64,' + 'A'.repeat(6000) };
  assert.equal(page.session.save(photo), 'without-photo');
  assert.deepEqual(JSON.parse(JSON.stringify(page.session.load())), snapshot());
});

test('blocked storage returns failures without throwing', () => {
  const page = browser(new Map(), { blocked: true });
  assert.equal(page.session.load(), null);
  assert.equal(page.session.save(snapshot()), 'failed');
  assert.equal(page.session.save(null), 'failed');
});
