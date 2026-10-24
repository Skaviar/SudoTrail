'use strict';

const $ = id => document.getElementById(id);
const board = $('board');
const cells = [];
let original = Array(81).fill(0);
let states = [];
let current = 0;
let busy = false;
let conflicts = new Set();
let puzzleLabel = '';

for (let index = 0; index < 81; index++) {
  const cell = document.createElement('div');
  cell.className = 'cell';
  const input = document.createElement('input');
  input.type = 'text';
  input.inputMode = 'numeric';
  input.autocomplete = 'off';
  input.setAttribute('aria-label', `Row ${Math.floor(index / 9) + 1}, column ${index % 9 + 1}`);
  input.addEventListener('input', () => {
    const digits = input.value.replace(/[^1-9]/g, '');
    original[index] = digits ? Number(digits.at(-1)) : 0;
    conflicts.clear();
    $('issues').hidden = true;
    message('Clues updated. Validate to check for conflicts.');
    render();
  });
  input.addEventListener('keydown', event => {
    const offsets = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -9, ArrowDown: 9 };
    if (Object.hasOwn(offsets, event.key)) {
      event.preventDefault();
      cells[Math.max(0, Math.min(80, index + offsets[event.key]))].input.focus();
    }
    if (!input.readOnly && (event.key === 'Delete' || event.key === 'Backspace' || event.key === '0')) {
      event.preventDefault();
      input.value = '';
      input.dispatchEvent(new Event('input'));
    }
  });
  input.addEventListener('focus', () => input.select());
  const marks = document.createElement('div');
  marks.className = 'candidate-grid';
  marks.setAttribute('aria-hidden', 'true');
  cell.append(marks, input);
  board.append(cell);
  cells.push({ cell, input, marks });
}

function message(text, kind = '') {
  $('message').textContent = text;
  $('message').className = `message ${kind}`;
}
function puzzleString() { return original.map(value => value || '.').join(''); }
function updateControls() {
  for (const id of ['random', 'validate', 'solve', 'clear', 'edit']) $(id).disabled = busy;
  $('validate').disabled = busy || states.length > 0;
  $('solve').disabled = busy || states.length > 0 || !original.some(Boolean);
  $('first').disabled = $('previous').disabled = busy || current === 0 || !states.length;
  $('last').disabled = $('next').disabled = busy || !states.length || current === states.length - 1;
  $('timeline').disabled = busy || states.length < 2;
  $('solve').textContent = busy ? 'Working…' : 'Solve step by step →';
  for (const { input } of cells) input.readOnly = busy || states.length > 0;
  board.setAttribute('aria-busy', String(busy));
}
function render() {
  const state = states[current];
  const step = state?.step;
  const removed = new Set((step?.eliminations || []).map(e => `${e.row},${e.col},${e.number}`));
  for (let index = 0; index < 81; index++) {
    const row = Math.floor(index / 9), col = index % 9;
    const { cell, input, marks } = cells[index];
    const value = state ? state.board[row][col] : original[index];
    input.value = value || '';
    input.setAttribute('aria-invalid', String(conflicts.has(index)));
    const placed = step?.number > 0 && step.row === row && step.col === col;
    cell.className = ['cell', original[index] ? 'clue' : '', conflicts.has(index) ? 'conflict' : '', placed ? 'placed' : ''].filter(Boolean).join(' ');
    marks.replaceChildren();
    if (state && !value && $('candidates').checked) {
      for (let digit = 1; digit <= 9; digit++) {
        const mark = document.createElement('span');
        const eliminated = removed.has(`${row},${col},${digit}`);
        if ((state.candidates[row][col] & (1 << digit)) || eliminated) mark.textContent = digit;
        if (eliminated) mark.className = 'removed';
        marks.append(mark);
      }
    }
  }
  $('empty-explanation').hidden = !!state;
  $('explanation').hidden = !state;
  $('edit').hidden = !state;
  $('mode').textContent = state ? 'FOLLOW THE LOGIC' : 'YOUR PUZZLE';
  $('board-title').textContent = state ? 'One deduction at a time' : puzzleLabel || 'Start with a blank slate';
  $('board-hint').textContent = state ? 'Use the arrows or slider to explore each step' : 'Select any cell · Type 1–9 · Delete to clear';
  $('step-count').textContent = state ? `${states.length - 1} deductions recorded` : 'Ready when you are';
  $('position').textContent = state ? `${current} / ${states.length - 1}` : '— / —';
  $('timeline').max = Math.max(0, states.length - 1);
  $('timeline').value = current;
  $('details').replaceChildren();
  if (state) {
    $('technique').textContent = step?.technique || 'Initial puzzle';
    $('step-title').textContent = step?.explanation || 'It all starts with your clues.';
    const details = step?.details || ['This is the board before any deductions. Move forward to see the first logical step.'];
    for (const text of details) {
      const item = document.createElement('li'); item.textContent = text; $('details').append(item);
    }
  }
  updateControls();
}
async function api(path, puzzle) {
  const response = await fetch(path, puzzle === undefined ? {} : {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ puzzle })
  });
  const data = await response.json();
  if (!response.ok) {
    if (data.issues) showIssues(data.issues);
    throw new Error(data.error || data.issues?.[0]?.message || 'The request could not be completed.');
  }
  return data;
}
async function perform(action) {
  if (busy) return;
  busy = true; updateControls();
  try { await action(); }
  catch (error) { message(error.message || 'Could not reach the server. Please try again.', 'error'); }
  finally { busy = false; render(); }
}
function showIssues(issues) {
  conflicts.clear(); $('issues').replaceChildren();
  for (const issue of issues) {
    for (const cell of issue.cells || []) conflicts.add(cell.row * 9 + cell.col);
    const item = document.createElement('li'); item.textContent = issue.message; $('issues').append(item);
  }
  $('issues').hidden = issues.length === 0;
}
$('validate').addEventListener('click', () => perform(async () => {
  const result = await api('/api/validate', puzzleString());
  showIssues(result.issues);
  message(result.valid ? 'No conflicts found. This checks the clues, not whether there is a unique solution.' : 'There are conflicts to resolve. Check the highlighted cells below.', result.valid ? 'success' : 'error');
}));
$('solve').addEventListener('click', () => perform(async () => {
  const result = await api('/api/solve', puzzleString());
  showIssues([]); states = result.states; current = 0;
  message(result.solved ? 'Solution found. Explore each deduction with the step controls.' : 'The solver reached its current technique limit. Explore the available steps; this does not mean the puzzle is unsolvable.', result.solved ? 'success' : '');
}));
$('random').addEventListener('click', () => perform(async () => {
  const result = await api('/api/puzzles/random');
  original = [...result.puzzle].map(ch => /[1-9]/.test(ch) ? Number(ch) : 0);
  states = []; current = 0; showIssues([]);
  puzzleLabel = `Puzzle #${result.id}${result.difficulty ? ` · Difficulty ${result.difficulty}` : ''}`;
  message('A fresh puzzle is ready. Validate it, make changes, or explore its solution.');
}));
$('clear').addEventListener('click', () => {
  original = Array(81).fill(0); states = []; current = 0; puzzleLabel = ''; showIssues([]);
  message('An empty board, endless possibilities. Add your clues to begin.'); render(); cells[0].input.focus();
});
$('edit').addEventListener('click', () => { states = []; current = 0; showIssues([]); message('Your original clues are restored. Select any cell to edit.'); render(); });
function navigate(index) { current = Math.max(0, Math.min(states.length - 1, index)); render(); }
$('first').addEventListener('click', () => navigate(0));
$('previous').addEventListener('click', () => navigate(current - 1));
$('next').addEventListener('click', () => navigate(current + 1));
$('last').addEventListener('click', () => navigate(states.length - 1));
$('timeline').addEventListener('input', event => navigate(Number(event.target.value)));
$('candidates').addEventListener('change', render);
render();
