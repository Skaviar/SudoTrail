'use strict';

const $ = id => document.getElementById(id);
const board = $('board');
const cells = [];
let original = Array(81).fill(0);
// Preserve puzzle provenance separately from the editable submission.
// null means a custom puzzle is still being entered.
let givenClues = null;
let states = [];
let current = 0;
let busy = false;
let solvePending = false;
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
    message('');
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
  $('validate').hidden = $('solve').hidden = solvePending || states.length > 0;
  for (const id of ['random', 'validate', 'solve', 'clear', 'edit', 'import-photo']) $(id).disabled = busy;
  $('validate').disabled = busy || states.length > 0;
  $('solve').disabled = busy || states.length > 0 || !original.some(Boolean);
  $('first').disabled = $('previous').disabled = busy || current === 0 || !states.length;
  $('last').disabled = $('next').disabled = busy || !states.length || current === states.length - 1;
  $('timeline').disabled = busy || states.length < 2;
  $('solve').textContent = busy ? 'Working…' : 'Solve';
  for (const { input } of cells) input.readOnly = busy || states.length > 0;
  board.setAttribute('aria-busy', String(busy));
}
function render() {
  const state = states[current];
  const enteringPlayback = !!state && !document.body.classList.contains('reviewing-steps');
  document.body.classList.toggle('reviewing-steps', !!state);
  const loaded = !!state || givenClues !== null || original.some(Boolean);
  document.body.classList.toggle('puzzle-loaded', loaded);
  $('guides-option').hidden = $('candidates-option').hidden = !loaded;
  $('random').hidden = $('import-photo').hidden = loaded;
  if (enteringPlayback && window.matchMedia('(max-width: 800px)').matches) {
    requestAnimationFrame(() => window.scrollTo({ top: 0, behavior: 'instant' }));
  }
  const step = state?.step;
  const removed = new Set((step?.eliminations || []).map(e => `${e.row},${e.col},${e.number}`));
  for (let index = 0; index < 81; index++) {
    const row = Math.floor(index / 9), col = index % 9;
    const { cell, input, marks } = cells[index];
    const value = state ? state.board[row][col] : original[index];
    input.value = value || '';
    input.setAttribute('aria-invalid', String(conflicts.has(index)));
    const placed = step?.number > 0 && step.row === row && step.col === col;
    cell.className = ['cell', value && (givenClues ? givenClues[index] === value : original[index] === value) ? 'clue' : '', conflicts.has(index) ? 'conflict' : '', placed ? 'placed' : ''].filter(Boolean).join(' ');
    marks.replaceChildren();
    if (state && !value && $('candidates').checked) {
      for (let digit = 1; digit <= 9; digit++) {
        const mark = document.createElement('span');
        const eliminated = removed.has(`${row},${col},${digit}`);
        if ((state.candidates[row][col] & (1 << digit)) || eliminated) mark.textContent = digit;
        if (eliminated) mark.className = 'removed';
        mark.dataset.digit = digit;
        marks.append(mark);
      }
    }
  }
  $('empty-explanation').hidden = !!state;
  $('explanation').hidden = !state;
  $('edit').hidden = !state;
  $('board-title').textContent = state ? 'Solution steps' : puzzleLabel || 'Enter a puzzle';
  $('board-hint').hidden = !!state || givenClues !== null;
  $('board-hint').textContent = 'Tap any cell to enter 1–9. Fill the clues in any order.';
  $('step-count').textContent = state ? `${states.length - 1} steps` : '';
  $('position').textContent = state ? `${current} / ${states.length - 1}` : '— / —';
  $('timeline').max = Math.max(0, states.length - 1);
  $('timeline').value = current;
  $('details').replaceChildren();
  if (state) {
    $('technique').textContent = step?.technique || 'Initial puzzle';
    $('step-title').textContent = step?.explanation || 'Original puzzle';
    const details = step?.details || ['Use Next to see the first move.'];
    for (const text of details) {
      const item = document.createElement('li'); item.textContent = text; $('details').append(item);
    }
  }
  renderVisuals(state, step);
  updateControls();
}
async function api(path, puzzle) {
  if (window.godokuEngine) {
    const response = await window.godokuEngine.request(path, puzzle);
    const data = response.body;
    if (response.status >= 400) {
      if (data.issues) showIssues(data.issues);
      throw new Error(data.error || data.issues?.[0]?.message || 'The browser solver could not process this puzzle.');
    }
    return data;
  }
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
async function perform(action, isSolve = false) {
  if (busy) return;
  busy = true; solvePending = isSolve; updateControls();
  try { await action(); }
  catch (error) { message(error.message || 'Could not reach the server. Please try again.', 'error'); }
  finally { busy = false; solvePending = false; render(); }
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
  message(result.valid ? 'No conflicts found.' : 'Fix the highlighted cells.', result.valid ? 'success' : 'error');
}));
$('solve').addEventListener('click', () => perform(async () => {
  const result = await api('/api/solve', puzzleString());
  if (givenClues === null) givenClues = [...original];
  showIssues([]); states = result.states; current = 0;
  message(result.solved ? '' : 'Stopped: no more supported moves.');
}, true));
$('random').addEventListener('click', () => perform(async () => {
  const result = await api('/api/puzzles/random');
  original = [...result.puzzle].map(ch => /[1-9]/.test(ch) ? Number(ch) : 0);
  givenClues = [...original];
  states = []; current = 0; showIssues([]);
  puzzleLabel = `Puzzle #${result.id}${result.difficulty ? ` · Difficulty ${result.difficulty}` : ''}`;
  message('');
}));
$('clear').addEventListener('click', () => {
  original = Array(81).fill(0); givenClues = null; states = []; current = 0; puzzleLabel = ''; showIssues([]);
  message(''); render(); cells[0].input.focus();
});
$('edit').addEventListener('click', () => { states = []; current = 0; showIssues([]); message(''); render(); });
function navigate(index) { current = Math.max(0, Math.min(states.length - 1, index)); render(); }
$('first').addEventListener('click', () => navigate(0));
$('previous').addEventListener('click', () => navigate(current - 1));
$('next').addEventListener('click', () => navigate(current + 1));
$('last').addEventListener('click', () => navigate(states.length - 1));
$('timeline').addEventListener('input', event => navigate(Number(event.target.value)));
$('candidates').addEventListener('change', render);
$('visuals').addEventListener('change', render);
new ResizeObserver(() => renderVisuals(states[current], states[current]?.step)).observe(board);
render();

// Read structured evidence from Go; placement propagation is a candidate-state diff.
function visualData(state, step) {
  if (!step) return { sources: [], digits: [], units: [], targets: [] };
  const visual = step.visual || {};
  const sources = visual.sources || (step.number > 0 ? [{ row: step.row, col: step.col }] : []);
  const targets = [...(step.eliminations || [])];
  const previous = states[current - 1];
  if (step.number > 0 && previous) {
    for (let row = 0; row < 9; row++) for (let col = 0; col < 9; col++) {
      if (!state.board[row][col] && (previous.candidates[row][col] & (1 << step.number)) && !(state.candidates[row][col] & (1 << step.number))) {
        targets.push({ row, col, number: step.number });
      }
    }
  }
  return { sources, digits: visual.digits || [step.number].filter(Boolean), units: visual.units || [], targets };
}
function renderVisuals(state, step) {
  const overlay = $('deduction-overlay');
  overlay.replaceChildren();
  for (const { cell, marks } of cells) {
    cell.classList.remove('focus-unit', 'source-cell', 'target-cell');
    for (const mark of marks.children) mark.classList.remove('evidence');
    // Remove propagation marks from a previous resize/render.
    for (const mark of marks.querySelectorAll('[data-propagated]')) { mark.textContent = ''; mark.classList.remove('removed'); delete mark.dataset.propagated; }
  }
  $('visual-summary').hidden = true;
  if (!state || !step || !$('visuals').checked) return;
  const { sources, digits, units, targets } = visualData(state, step);
  const bounds = board.getBoundingClientRect();
  overlay.setAttribute('viewBox', `0 0 ${bounds.width} ${bounds.height}`);
  const svg = (tag, attributes) => {
    const element = document.createElementNS('http://www.w3.org/2000/svg', tag);
    for (const [key, value] of Object.entries(attributes)) element.setAttribute(key, value);
    overlay.append(element); return element;
  };
  const cellAt = p => cells[p.row * 9 + p.col];
  const point = (p, digit) => {
    const entry = cellAt(p);
    const mark = $('candidates').checked && !state.board[p.row][p.col] && digit ? entry.marks.querySelector(`[data-digit="${digit}"]`) : null;
    const rect = (mark || entry.cell).getBoundingClientRect();
    return { x: rect.left - bounds.left + rect.width / 2, y: rect.top - bounds.top + rect.height / 2 };
  };
  for (const unit of units) {
    const members = [];
    for (let row = 0; row < 9; row++) for (let col = 0; col < 9; col++) {
      if ((unit.kind === 'row' && row === unit.index) || (unit.kind === 'column' && col === unit.index) || (unit.kind === 'box' && Math.floor(row / 3) * 3 + Math.floor(col / 3) === unit.index)) members.push({ row, col });
    }
    if (!members.length) continue;
    for (const member of members) cellAt(member).cell.classList.add('focus-unit');
    const start = cellAt(members[0]).cell.getBoundingClientRect(), end = cellAt(members.at(-1)).cell.getBoundingClientRect();
    svg('rect', { x: start.left - bounds.left + 3, y: start.top - bounds.top + 3, width: end.right - start.left - 6, height: end.bottom - start.top - 6, rx: 5, class: 'guide-unit' });
  }
  for (const source of sources) {
    const entry = cellAt(source); entry.cell.classList.add('source-cell');
    for (const digit of digits) {
      const mark = entry.marks.querySelector(`[data-digit="${digit}"]`);
      if (mark?.textContent && !mark.classList.contains('removed')) mark.classList.add('evidence');
    }
  }
  // Locked candidates get one clear axis through their source cells.
  if (sources.length > 1 && digits.length === 1) {
    const points = sources.map(source => point(source, digits[0]));
    if (sources.every(p => p.row === sources[0].row) || sources.every(p => p.col === sources[0].col)) {
      const a = points[0], b = points.at(-1);
      svg('line', { x1: a.x, y1: a.y, x2: b.x, y2: b.y, class: 'guide-axis' });
    }
  }
  for (const target of targets) {
    const entry = cellAt(target); entry.cell.classList.add('target-cell');
    const mark = entry.marks.querySelector(`[data-digit="${target.number}"]`);
    if (mark && !mark.textContent) { mark.textContent = target.number; mark.classList.add('removed'); mark.dataset.propagated = 'true'; }
    const end = point(target, target.number);
    const sameCell = sources.some(source => source.row === target.row && source.col === target.col);
    if (!sameCell && sources.length) {
      // One nearest source per target avoids a web of crossing connections.
      const candidates = sources.map(source => point(source, target.number));
      const start = candidates.reduce((a, b) => Math.hypot(a.x - end.x, a.y - end.y) < Math.hypot(b.x - end.x, b.y - end.y) ? a : b);
      svg('line', { x1: start.x, y1: start.y, x2: end.x, y2: end.y, class: 'guide-link' });
    }
    const radius = $('candidates').checked ? Math.max(5, bounds.width / 70) : bounds.width / 34;
    svg('circle', { cx: end.x, cy: end.y, r: radius, class: 'guide-ring' });
    if (!$('candidates').checked) svg('path', { d: `M${end.x - 4},${end.y - 4}l8,8m0,-8l-8,8`, class: 'guide-cross' });
  }
  const unitNames = units.map(u => `${u.kind === 'column' ? 'Column' : u.kind === 'box' ? 'Box' : 'Row'} ${u.index + 1}`);
  const text = [];
  if (unitNames.length) text.push(unitNames.join(' + '));
  if (targets.length) text.push(`${targets.length} candidate${targets.length === 1 ? '' : 's'} removed${step.number > 0 ? ' from peers' : ''}`);
  $('visual-summary').textContent = text.join(' · ');
  $('visual-summary').hidden = !text.length;
}

// Photo import supplies reviewed clues; correction and solving use the normal board.
document.addEventListener('godoku:photo-import', event => {
  if (busy) return;
  original = [...event.detail.values];
  givenClues = [...original];
  states = []; current = 0; puzzleLabel = 'Puzzle from your photo';
  showIssues([]);
  message('Photo imported. Check the clues, then validate.');
  render();
});

// On phones, keep the explanation compact while navigation remains in reach.
const mobileLayout = window.matchMedia('(max-width: 800px)');
function configureExplanation() {
  $('reasoning-details').open = false;
}
mobileLayout.addEventListener('change', configureExplanation);
configureExplanation();

// Fit the entire board inside its available slot, including browser-toolbar and
// dock size changes. The hint lives inside the optional reasoning details.
const boardStage = document.querySelector('.board-stage');
function fitBoard() {
  const compact = window.matchMedia('(max-width: 800px)').matches;
  const reviewing = document.body.classList.contains('reviewing-steps');
  if (compact && reviewing) {
    const size = Math.floor(Math.min(boardStage.clientWidth, boardStage.clientHeight));
    boardStage.style.setProperty('--board-space', `${Math.max(0, size)}px`);
  } else boardStage.style.removeProperty('--board-space');
}
const layoutObserver = new ResizeObserver(entries => {
  const dock = document.querySelector('.playback');
  if (entries.some(entry => entry.target === dock)) {
    document.documentElement.style.setProperty('--dock-height', `${Math.ceil(dock.getBoundingClientRect().height)}px`);
  }
  fitBoard();
});
layoutObserver.observe(boardStage);
layoutObserver.observe(document.querySelector('.playback'));
window.visualViewport?.addEventListener('resize', fitBoard);
