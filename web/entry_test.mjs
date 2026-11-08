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
      blur() { if (document.activeElement === this) document.activeElement = null; },
      select() {}, selectedOptions: [{}]
    };
  }
  const document = {
    getElementById(id) { if (!elements.has(id)) elements.set(id, element()); return elements.get(id); },
    listeners: {},
    createElement: element,
    addEventListener(name, callback) { this.listeners[name] = callback; },
    querySelectorAll() { return []; }
  };
  const context = vm.createContext({
    document, window: {
      godokuSession: { load: () => null }, godokuTraining: {},
      matchMedia: () => ({ matches: mobile }), addEventListener() {}
    }, setTimeout() {}, clearTimeout() {}
  });
  // Run the real entry handlers and control logic without playback or API calls.
  vm.runInContext(source.slice(0, source.indexOf('// Manual entry and photo review')) +
    source.slice(source.indexOf('function updateControls()'), source.indexOf('function render()')) +
    source.slice(source.indexOf('function puzzleString()'), source.indexOf('function renderTechniqueLink')) +
    source.slice(source.indexOf('async function requestHint()'), source.indexOf("$('hint').addEventListener")) + `
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
  assert.equal(run('cells[0].input.type'), 'button');
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


test('a hint guides without filling any cell and keeps the board editable', async () => {
  const run = setup(true);
  run(`givenClues = Array(81).fill(0); givenClues[0] = 5; original = [...givenClues];
    async function api(path) {
      if (path !== '/api/hint') throw Error('unexpected endpoint');
      return {status: 'hint', focus: {row: 0, col: 1}, technique: 'Hidden Single', guidance: 'Check where each missing digit can fit.'};
    }`);
  await run('requestHint();');
  run('updateControls();');
  assert.equal(run('original.filter(Boolean).length'), 1);
  assert.equal(run('states.length'), 0);
  assert.equal(run('canEditCell(1)'), true);
  assert.equal(run('selectedCell'), 1);
  assert.equal(run('editHistory.length'), 0);
  assert.equal(run(`$('hint-text').textContent`), 'Check where each missing digit can fit.');
  assert.equal(run(`$('hint').textContent`), 'Hint');
  assert.equal(run('typeof applyHint'), 'undefined');
});

test('editing invalidates guidance and terminal hints do not change the board', async () => {
  const run = setup();
  run(`async function api() { return {status: 'hint', technique: 'Naked Pair', guidance: 'Compare candidates.'}; }`);
  await run('requestHint()');
  assert.equal(run('selectedCell'), -1);
  run('editCell(2, 4);');
  assert.equal(run('activeHint'), null);
  run(`api = async () => ({status: 'solved'});`);
  await run('requestHint()');
  assert.equal(run('activeHint'), null);
  assert.equal(run('original[2]'), 4);
});


test('manual digits reset highlights and keypad until a cell is selected again', () => {
  for (const mobile of [false, true]) {
    const run = setup(mobile);
    run(`$('number-pad').children = Array.from({length: 9}, () => ({disabled: false}));`);
    run('cells[1].input.focus(); activeHint = {focus: {row: 0, col: 1}};');
    assert.equal(run('highlightedCell'), 1);
    run('editCell(1, 4);');
    assert.equal(run('highlightedCell'), -1);
    assert.equal(run('activeHint'), null);
    assert.equal(run('selectedCell'), -1);
    assert.equal(run(`$('number-pad').children.every(button => button.disabled)`), true);
    run('cells[1].input.focus();');
    assert.equal(run('selectedCell'), 1);
    assert.equal(run(`$('number-pad').children.some(button => button.disabled)`), false);
    run('cells[2].input.focus();');
    assert.equal(run('highlightedCell'), 2);
  }
});


test('outside taps clear selection and reset the keypad while cell controls preserve it', () => {
  const run = setup(true);
  run(`$('number-pad').children = Array.from({length: 9}, () => ({disabled: false})); cells[1].input.focus();`);
  run(`document.listeners.pointerdown({target: {closest: () => ({})}});`);
  assert.equal(run('selectedCell'), 1);
  assert.equal(run('highlightedCell'), 1);
  run(`document.listeners.pointerdown({target: {closest: () => null}});`);
  assert.equal(run('selectedCell'), -1);
  assert.equal(run('highlightedCell'), -1);
  assert.equal(run('document.activeElement'), null);
  assert.equal(run(`$('number-pad').children.every(button => button.disabled)`), true);
  run('cells[2].input.focus();');
  assert.equal(run('selectedCell'), 2);
  assert.equal(run(`$('number-pad').children.some(button => button.disabled)`), false);
});


test('tapping a cell selects it even when the browser does not focus buttons', () => {
  const run = setup(true);
  run(`$('number-pad').children = Array.from({length: 9}, () => ({disabled: false}));`);
  run('cells[1].input.listeners.click();');
  assert.equal(run('selectedCell'), 1);
  assert.equal(run('highlightedCell'), 1);
  assert.equal(run(`$('number-pad').children.some(button => button.disabled)`), false);
  run(`document.listeners.pointerdown({target: {closest: () => null}});`);
  assert.equal(run('highlightedCell'), -1);
  run('cells[1].input.listeners.click();');
  assert.equal(run('highlightedCell'), 1);
  run('editCell(1, 4);');
  assert.equal(run('highlightedCell'), -1);
});
