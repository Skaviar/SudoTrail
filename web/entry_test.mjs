import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

const source = readFileSync(new URL('./app.js', import.meta.url), 'utf8');
function setup(mobile = false) {
  const elements = new Map();
  function element() {
    return {
      value: '', hidden: false, disabled: false, dataset: {}, children: [], listeners: {},
      classList: { toggle() {} }, setAttribute() {},
      append(...children) { this.children.push(...children); },
      addEventListener(name, callback) { this.listeners[name] = callback; },
      querySelectorAll() { return this.children; },
      focus() { document.activeElement = this; this.listeners.focus?.(); },
      select() {}, selectedOptions: [{}]
    };
  }
  const document = {
    getElementById(id) { if (!elements.has(id)) elements.set(id, element()); return elements.get(id); },
    createElement: element, addEventListener() {}, querySelectorAll() { return []; }
  };
  const context = vm.createContext({
    document, window: {
      godokuSession: { load: () => null }, godokuTraining: {},
      matchMedia: () => ({ matches: mobile }), addEventListener() {}
    }, setTimeout() {}, clearTimeout() {}
  });
  // Run the real entry handlers and control logic without playback or API calls.
  vm.runInContext(source.slice(0, source.indexOf('// Manual entry and photo review')) +
    source.slice(source.indexOf('function updateControls()'), source.indexOf('function render()')) + `
    function showIssues() {}
    function message() {}
    function scheduleValidation() {}
    function cancelValidation() {}
    function render() { updateControls(); }
    `, context);
  return code => vm.runInContext(code, context);
}

test('loaded clues reject entry, clearing and synthetic input; blank cells remain editable', () => {
  const run = setup();
  run('givenClues = Array(81).fill(0); givenClues[0] = 5; original = [...givenClues]; updateControls();');
  assert.equal(run('cells[0].input.readOnly'), true);
  assert.equal(run('cells[1].input.readOnly'), false);
  run('editCell(0, 9); editCell(0, 0); cells[0].input.value = "7"; cells[0].input.listeners.input({data: "7"});');
  assert.equal(run('original[0]'), 5);
  run('editCell(1, 4);');
  assert.equal(run('original[1]'), 4);
});

test('undo restores replacements and cleared values in order and stops at the loaded puzzle', () => {
  const run = setup();
  run('editCell(1, 4); editCell(1, 8); editCell(1, 0); undoEdit();');
  assert.equal(run('original[1]'), 8);
  run('undoEdit();');
  assert.equal(run('original[1]'), 4);
  run('undoEdit(); undoEdit();');
  assert.equal(run('original[1]'), 0);
  assert.equal(run('editHistory.length'), 0);
  run('editCell(1, 3); resetEdits();');
  assert.equal(run('editHistory.length'), 0);
});

test('mobile selection suppresses the OS keyboard while allowing entry and physical keys', () => {
  const run = setup(true);
  run('updateControls(); cells[0].input.focus();');
  assert.equal(run('cells[0].input.readOnly'), true);
  assert.equal(run('cells[0].input.inputMode'), 'none');
  assert.equal(run('selectedCell'), 0);
  run('editCell(selectedCell, 6);');
  assert.equal(run('original[0]'), 6);
  run('cells[1].input.listeners.keydown({key: "7", preventDefault() {}});');
  assert.equal(run('original[1]'), 7);
});

test('playback and pending solver work block all edits and undo', () => {
  const run = setup();
  run('editCell(0, 4); states = [{}]; editCell(0, 9); undoEdit();');
  assert.equal(run('original[0]'), 4);
  run('states = []; solvePending = true; editCell(0, 9); undoEdit();');
  assert.equal(run('original[0]'), 4);
});
