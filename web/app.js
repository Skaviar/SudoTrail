'use strict';

const $ = id => document.getElementById(id);
const board = $('board');
const cells = [];
const savedSession = window.godokuSession.load();
const training = window.godokuTraining;
let original = savedSession?.original || Array(81).fill(0);
// Preserve puzzle provenance separately from the editable submission.
// null means a custom puzzle is still being entered.
let givenClues = savedSession?.givenClues || null;
let transcribing = savedSession?.transcribing ?? true;
let referencePhoto = savedSession?.referencePhoto || '';
let states = [];
let current = 0;
let restoringStep = savedSession?.reviewing ? savedSession.current : null;
let busy = restoringStep !== null;
let solvePending = false;
let conflicts = new Set();
let puzzleLabel = savedSession?.puzzleLabel || '';
let saveTimer;
let validationTimer;
let validationRevision = 0;
let hasSavedPuzzle = savedSession !== null;
let proofStep = null;
let selectedProof = 0;
let entryMode = savedSession !== null;
let selectedDifficulty = savedSession?.selectedDifficulty || 'easy';
let selectedTechnique = savedSession?.selectedTechnique || '';
let practiceFocus = savedSession?.practiceFocus || '';
let challengePending = savedSession?.challengePending === true;
let practiceTarget = -1;
let lastPuzzleID = savedSession?.lastPuzzleID || '';
let practiceCatalog = [];
let selectedCell = -1;
let highlightedCell = -1;
let editHistory = [];
let activeHint = null;
let hintPage = 1;
const keypadLayout = window.matchMedia('(max-width: 800px), (pointer: coarse)');
function canEditCell(index) {
  return index >= 0 && index < 81 && !busy && !solvePending && !states.length && !givenClues?.[index];
}
function editCell(index, value) {
  if (!canEditCell(index)) return;
  activeHint = null;
  if (value > 0) {
    highlightedCell = -1;
    selectedCell = -1;
    if (document.activeElement === cells[index].input) cells[index].input.blur();
  }
  if (original[index] === value) { render(); return; }
  editHistory.push({ index, value: original[index] });
  original[index] = value;
  activeHint = null;
  showIssues([]); message(''); render(); scheduleValidation();
}
function undoEdit() {
  if (busy || solvePending || states.length || !editHistory.length) return;
  const edit = editHistory.pop();
  // A newly loaded puzzle never inherits edits from the previous one.
  if (givenClues?.[edit.index]) return;
  original[edit.index] = edit.value;
  activeHint = null;
  selectedCell = edit.index;
  highlightedCell = edit.index;
  showIssues([]); message(''); render(); scheduleValidation();
  cells[selectedCell].input.focus();
}
function resetEdits() { editHistory = []; selectedCell = -1; highlightedCell = -1; activeHint = null; }

// Keep selection while using the cell controls; other taps end cell entry.
document.addEventListener('pointerdown', event => {
  if (event.target.closest?.('#board .cell, #number-pad button, #clear-cell, #undo')) return;
  if (selectedCell < 0 && highlightedCell < 0) return;
  selectedCell = -1;
  highlightedCell = -1;
  for (const { cell, input } of cells) {
    cell.classList.toggle('selected', false);
    if (document.activeElement === input) input.blur();
  }
  updateControls();
});

$('candidates').checked = savedSession?.candidates ?? true;
$('visuals').checked = savedSession?.guides ?? true;
if (referencePhoto) $('reference-image').src = referencePhoto;

function scheduleSessionSave() {
  clearTimeout(saveTimer);
  saveTimer = setTimeout(saveSession, 150);
}
function saveSession(clearEmpty = false) {
  clearTimeout(saveTimer);
  const hasPuzzle = original.some(Boolean) || givenClues !== null || states.length > 0;
  if (!hasPuzzle && !hasSavedPuzzle && clearEmpty !== true) {
    $('save-status').hidden = true;
    return;
  }
  const snapshot = hasPuzzle ? {
    original, givenClues, transcribing, referencePhoto, puzzleLabel,
    selectedDifficulty, selectedTechnique, practiceFocus, challengePending, lastPuzzleID,
    reviewing: states.length > 0 || restoringStep !== null,
    current: restoringStep ?? current,
    candidates: $('candidates').checked, guides: $('visuals').checked
  } : null;
  const result = window.godokuSession.save(snapshot);
  if (result !== 'failed') hasSavedPuzzle = hasPuzzle;
  const text = result === 'failed' ? (hasPuzzle ? 'This browser could not save your puzzle. Keep this page open to retain your progress.' : 'This browser could not remove the saved puzzle.') :
    result === 'without-photo' ? 'Puzzle saved. The reference photo could not fit in browser storage.' : '';
  $('save-status').textContent = text;
  $('save-status').hidden = !text;
}
// Flush pending edits when a phone backgrounds the page or the tab closes.
window.addEventListener('pagehide', saveSession);
document.addEventListener('visibilitychange', () => { if (document.hidden) saveSession(); });

for (let index = 0; index < 81; index++) {
  const cell = document.createElement('div');
  cell.className = 'cell';
  const input = document.createElement('input');
  // Mobile cells select a keypad target, so they need no browser text services.
  input.type = keypadLayout.matches ? 'button' : 'text';
  input.inputMode = 'numeric';
  input.autocomplete = 'off';
  input.name = `sudoku-cell-${index}`;
  input.spellcheck = false;
  input.setAttribute('autocorrect', 'off');
  input.setAttribute('autocapitalize', 'off');
  input.setAttribute('aria-label', `Row ${Math.floor(index / 9) + 1}, column ${index % 9 + 1}`);
  input.addEventListener('input', event => {
    if (!canEditCell(index)) {
      render();
      return;
    }
    if (event.isComposing) return;
    const enteredDigit = /[1-9]/.test(event.data ?? input.value);
    const wasFocused = document.activeElement === input;
    const insertedText = event.inputType?.startsWith('insert') || event.data != null;
    if (transcribing && wasFocused && insertedText && !enteredDigit) {
      input.value = original[index] || '';
      focusNextEmptyCell(index);
      return;
    }
    const digits = input.value.replace(/[^1-9]/g, '');
    editCell(index, digits ? Number(digits.at(-1)) : 0);
    if (transcribing && digits && enteredDigit && wasFocused) focusNextEmptyCell(index);
  });
  input.addEventListener('keydown', event => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'z' && !event.shiftKey) {
      event.preventDefault(); undoEdit(); return;
    }
    // Read-only mobile inputs suppress the OS keyboard; physical keys still work.
    if (keypadLayout.matches && canEditCell(index) && !event.ctrlKey && !event.metaKey && !event.altKey) {
      if (/^[1-9]$/.test(event.key)) {
        event.preventDefault(); editCell(index, Number(event.key));
        if (transcribing) focusNextEmptyCell(index);
        return;
      }
      if (['Backspace', 'Delete', '0'].includes(event.key)) {
        event.preventDefault(); editCell(index, 0); return;
      }
    }
    handleGridKey(event, index, cells, original, transcribing && canEditCell(index));
  });
  const selectCell = () => {
    selectedCell = index;
    highlightedCell = index;
    cells.forEach((entry, i) => entry.cell.classList.toggle('selected', i === index));
    updateControls();
    if (!input.readOnly) input.select();
  };
  input.addEventListener('focus', selectCell);
  // Touch browsers may activate a button without moving keyboard focus.
  input.addEventListener('click', selectCell);
  input.addEventListener('compositionstart', cancelValidation);
  const marks = document.createElement('div');
  marks.className = 'candidate-grid';
  marks.setAttribute('aria-hidden', 'true');
  cell.append(marks, input);
  board.append(cell);
  cells.push({ cell, input, marks });
}

// Advance in reading order, skipping every filled cell. Wrap to earlier blanks
// so entering a puzzle out of order still works; a full board keeps focus.
function focusNextEmptyCell(index, values = original, entries = cells) {
  for (let offset = 1; offset < entries.length; offset++) {
    const next = (index + offset) % entries.length;
    if (values[next] === 0) {
      entries[next].input.focus();
      return;
    }
  }
}

// Manual entry and photo review share navigation, skipping, and deletion.
function handleGridKey(event, index, entries, values, advance) {
  if (event.isComposing) return;
  const input = entries[index].input;
  if (advance && !input.readOnly && !event.ctrlKey && !event.metaKey && !event.altKey &&
      (event.key === 'Enter' || (event.key.length === 1 && !/^[1-9]$/.test(event.key)))) {
    event.preventDefault();
    focusNextEmptyCell(index, values, entries);
    return;
  }
  const offsets = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -9, ArrowDown: 9 };
  if (Object.hasOwn(offsets, event.key)) {
    event.preventDefault();
    entries[Math.max(0, Math.min(entries.length - 1, index + offsets[event.key]))].input.focus();
  }
  if (!input.readOnly && ['Delete', 'Backspace', '0'].includes(event.key)) {
    event.preventDefault();
    input.value = '';
    input.dispatchEvent(new Event('input'));
  }
}

function message(text, kind = '') {
  $('message').textContent = text;
  $('message').className = `message ${kind}`;
}
function puzzleString() { return original.map(value => value || '.').join(''); }
function renderTechniqueLink(id, technique) {
  const link = $(id), url = training.guideUrl(technique);
  link.hidden = !url;
  if (!url) {
    link.removeAttribute('href'); link.removeAttribute('aria-label'); link.removeAttribute('title');
    return;
  }
  const site = url.startsWith('https://www.sudokuwiki.org/') ? 'SudokuWiki' : 'HoDoKu';
  link.href = url;
  link.setAttribute('aria-label', `Learn ${technique} on ${site} (opens in a new tab)`);
  link.title = `${site}: ${technique} (opens in a new tab)`;
}
function updateControls() {
  $('solve').hidden = solvePending || states.length > 0;
  $('hint').hidden = states.length > 0;
  for (const id of ['random', 'solve', 'clear', 'edit', 'import-photo', 'enter-puzzle', 'reveal-hint', 'all-steps']) $(id).disabled = busy;
  $('random').disabled = busy || !practiceCatalog.length || !$('technique-focus').selectedOptions[0] || $('technique-focus').selectedOptions[0].disabled;
  $('technique-focus').disabled = busy || !practiceCatalog.length;
  for (const radio of document.querySelectorAll('[name="difficulty"]')) radio.disabled = busy;
  $('solve').disabled = busy || states.length > 0 || !original.some(Boolean) || conflicts.size > 0;
  $('hint').disabled = $('solve').disabled;
  $('hint').textContent = 'Hint';
  $('hint-card').hidden = !activeHint || states.length > 0;
  $('hint').setAttribute('aria-expanded', String(!$('hint-card').hidden));
  $('dismiss-hint').disabled = busy;
  if (activeHint) {
    $('hint-title').textContent = activeHint.technique;
    $('hint-position').textContent = `${hintPage}/2`;
    $('hint-text').textContent = hintPage === 1 ? activeHint.guidance : activeHint.unitGuidance;
    $('hint-page').textContent = hintPage === 1 ? 'Next →' : '← Back';
    $('hint-page').disabled = busy;
  }
  if (!$('hint-card').hidden && !$('hint-card').open) $('hint-card').showModal();
  else if ($('hint-card').hidden && $('hint-card').open) $('hint-card').close();
  $('first').disabled = $('previous').disabled = busy || current === 0 || !states.length;
  $('last').disabled = $('next').disabled = busy || !states.length || current === states.length - 1;
  $('timeline').disabled = busy || states.length < 2;
  $('solve').textContent = busy ? 'Working…' : 'Solve step by step';
  cells.forEach(({ input }, index) => {
    input.type = keypadLayout.matches ? 'button' : 'text';
    input.readOnly = !canEditCell(index) || keypadLayout.matches;
    input.inputMode = keypadLayout.matches ? 'none' : 'numeric';
    input.setAttribute('aria-readonly', String(!canEditCell(index)));
  });
  $('cell-controls').hidden = !entryMode || states.length > 0;
  $('number-pad').hidden = !keypadLayout.matches;
  for (const button of $('number-pad').querySelectorAll('button')) button.disabled = !canEditCell(selectedCell);
  $('clear-cell').disabled = !canEditCell(selectedCell) || !original[selectedCell];
  $('undo').disabled = busy || solvePending || states.length > 0 || !editHistory.length;
  board.setAttribute('aria-busy', String(busy));
}
function render() {
  const state = states[current];
  const enteringPlayback = !!state && !document.body.classList.contains('reviewing-steps');
  document.body.classList.toggle('reviewing-steps', !!state);
  const choosing = !entryMode && !state;
  $('puzzle-chooser').hidden = !choosing;
  document.querySelector('.workspace').hidden = choosing;
  const loaded = !choosing;
  document.body.classList.toggle('puzzle-loaded', loaded);
  document.body.classList.toggle('practice-challenge', !!state && challengePending);
  $('guides-option').hidden = $('candidates-option').hidden = !state;
  const showPhoto = !!referencePhoto && !state;
  $('photo-reference').hidden = !showPhoto;
  document.body.classList.toggle('photo-entry', showPhoto);
  if (enteringPlayback && window.matchMedia('(max-width: 800px)').matches) {
    requestAnimationFrame(() => window.scrollTo({ top: 0, behavior: 'instant' }));
  }
  const step = challengePending ? null : state?.step;
  if (proofStep !== step) { proofStep = step; selectedProof = 0; }
  const removed = new Set((step?.eliminations || []).map(e => `${e.row},${e.col},${e.number}`));
  for (let index = 0; index < 81; index++) {
    const row = Math.floor(index / 9), col = index % 9;
    const { cell, input, marks } = cells[index];
    const value = state ? state.board[row][col] : original[index];
    input.value = value || '';
    input.setAttribute('aria-invalid', String(conflicts.has(index)));
    const placed = step?.number > 0 && step.row === row && step.col === col;
    cell.className = ['cell', index === highlightedCell ? 'selected' : '', value && (givenClues ? givenClues[index] === value : original[index] === value) ? 'clue' : '', conflicts.has(index) ? 'conflict' : '', placed ? 'placed' : ''].filter(Boolean).join(' ');
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
  $('board-hint').hidden = !!state;
  $('board-hint').textContent = keypadLayout.matches
    ? givenClues === null ? 'Select a cell, then tap a number to enter your puzzle.'
      : 'Select a cell, then tap a number. Original clues are locked.'
    : transcribing ? 'Type 1–9 for clues; 0, space or another character skips a blank.'
    : 'Fill the empty cells. Original clues are locked.';
  $('step-count').textContent = state ? `${states.length - 1} steps` : '';
  if (state && practiceFocus) $('step-count').textContent = `${practiceFocus} practice`;
  $('position').textContent = state ? `${current} / ${states.length - 1}` : '— / —';
  $('timeline').max = Math.max(0, states.length - 1);
  $('timeline').value = current;
  $('details').replaceChildren();
  $('challenge-tip').hidden = !challengePending || !state;
  $('challenge-tip').textContent = challengePending ? training.describe(practiceFocus) : '';
  $('reasoning-details').hidden = !!state && challengePending;
  renderTechniqueLink('technique-guide', state ? challengePending ? practiceFocus : step?.technique : null);
  $('reveal-hint').hidden = $('all-steps').hidden = !state || !challengePending;
  const proofs = step?.visual?.proofs || [];
  $('proof-choice').hidden = proofs.length < 2;
  $('proof-branch').replaceChildren();
  for (let i = 0; i < proofs.length; i++) {
    const option = document.createElement('option');
    option.value = i; option.textContent = `Branch ${i + 1} of ${proofs.length}`;
    option.selected = i === selectedProof; $('proof-branch').append(option);
  }
  if (state) {
    $('technique').textContent = challengePending ? 'Your turn' : step?.technique || 'Initial puzzle';
    $('step-title').textContent = challengePending ? `Find the ${practiceFocus}.` : step?.explanation || 'Original puzzle';
    const branch = proofs[selectedProof];
    const details = branch ? branch.nodes.map((node, i) =>
      `${i + 1}. R${node.row + 1}C${node.col + 1} ${node.true ? '=' : '≠'} ${node.number}: ${node.reason}${node.parents?.length ? ` (from ${node.parents.map(p => p + 1).join(', ')})` : ''}.`
    ) : step?.details || ['Use Next to see the first move.'];
    if (branch?.contradiction) details.push(branch.contradiction);
    for (const text of details) {
      const item = document.createElement('li'); item.textContent = text; $('details').append(item);
    }
  }
  renderVisuals(state, step);
  updateControls();
  scheduleSessionSave();
}
async function api(path, puzzle, { reportIssues = true } = {}) {
  if (window.godokuEngine) {
    const response = await window.godokuEngine.request(path, puzzle);
    const data = response.body;
    if (response.status >= 400) {
      if (reportIssues && data.issues) showIssues(data.issues);
      throw new Error(data.error || data.issues?.[0]?.message || 'The browser solver could not process this puzzle.');
    }
    return data;
  }
  const response = await fetch(path, puzzle === undefined ? {} : {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ puzzle })
  });
  const data = await response.json();
  if (!response.ok) {
    if (reportIssues && data.issues) showIssues(data.issues);
    throw new Error(data.error || data.issues?.[0]?.message || 'The request could not be completed.');
  }
  return data;
}
async function perform(action, isSolve = false) {
  if (busy) return;
  cancelValidation();
  busy = true; solvePending = isSolve; updateControls();
  try { await action(); }
  catch (error) {
    const text = error.message || 'Could not reach the solver. Please try again.';
    message(text, 'error'); $('chooser-error').textContent = text; $('chooser-error').hidden = entryMode;
  }
  finally { busy = false; solvePending = false; render(); }
}
function showIssues(issues) {
  conflicts.clear(); $('issues').replaceChildren();
  for (const issue of issues) {
    for (const cell of issue.cells || []) conflicts.add(cell.row * 9 + cell.col);
    const item = document.createElement('li'); item.textContent = issue.message; $('issues').append(item);
  }
  $('issues').hidden = issues.length === 0;
  // Background checks update the highlights without touching the active input.
  cells.forEach(({ cell, input }, index) => {
    cell.classList.toggle('conflict', conflicts.has(index));
    input.setAttribute('aria-invalid', String(conflicts.has(index)));
  });
}

function cancelValidation() {
  clearTimeout(validationTimer);
  validationRevision++;
}
function scheduleValidation() {
  cancelValidation();
  if (!entryMode || states.length || !original.some(Boolean)) return;
  const revision = validationRevision, puzzle = puzzleString();
  const isCurrent = () => revision === validationRevision && entryMode && !states.length && !busy && puzzle === puzzleString();
  // Coalesce rapid edits and ignore replies for a different board or mode.
  validationTimer = setTimeout(async () => {
    if (!isCurrent()) return;
    try {
      const result = await api('/api/validate', puzzle, { reportIssues: false });
      if (!isCurrent()) return;
      showIssues(result.issues);
      message(result.valid ? '' : 'Fix the highlighted cells.', result.valid ? '' : 'error');
      updateControls();
    } catch {
      if (isCurrent()) message('Automatic check unavailable. Edit a cell to retry.', 'error');
    }
  }, 150);
}
async function requestHint() {
  const result = await api('/api/hint', puzzleString());
  showIssues([]);
  if (result.status !== 'hint') {
    activeHint = null;
    message(result.status === 'solved' ? 'Puzzle complete!' : 'No supported hint is available from this position. Check your entries or try another move.');
    return;
  }
  if (givenClues === null) { givenClues = [...original]; resetEdits(); }
  transcribing = false;
  activeHint = result;
  hintPage = 1;
  if (activeHint.focus) {
    selectedCell = activeHint.focus.row * 9 + activeHint.focus.col;
    highlightedCell = selectedCell;
  }
  message('');

}
$('hint').addEventListener('click', () => perform(requestHint));
$('hint-page').addEventListener('click', () => {
  if (!activeHint || busy) return;
  hintPage = hintPage === 1 ? 2 : 1;
  updateControls();
});
$('dismiss-hint').addEventListener('click', () => { activeHint = null; render(); });
$('hint-card').addEventListener('close', () => {
  if (activeHint && !$('hint-card').open) { activeHint = null; render(); }
});
$('hint-card').addEventListener('click', event => {
  if (event.target !== $('hint-card')) return;
  const bounds = $('hint-card').getBoundingClientRect();
  if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) {
    activeHint = null; render();
  }
});
$('solve').addEventListener('click', () => perform(async () => {
  transcribing = false;
  const result = await api('/api/solve', puzzleString());
  if (givenClues === null) { givenClues = [...original]; resetEdits(); }
  activeHint = null;
  showIssues([]); states = result.states; current = 0;
  setUpChallenge();
  message(result.solved ? '' : 'Stopped: no more supported moves.');
}, true));
$('random').addEventListener('click', () => perform(async () => {
  const query = new URLSearchParams({ source: 'practice', difficulty: selectedDifficulty, technique: selectedTechnique, exclude: lastPuzzleID });
  const result = await api(`/api/puzzles/random?${query}`);
  resetEdits();
  original = [...result.puzzle].map(ch => /[1-9]/.test(ch) ? Number(ch) : 0);
  givenClues = [...original];
  transcribing = false; referencePhoto = ''; $('reference-image').removeAttribute('src');
  states = []; current = 0; challengePending = false; practiceTarget = -1; showIssues([]);
  entryMode = true; practiceFocus = selectedTechnique; lastPuzzleID = result.id;
  puzzleLabel = `${training.levels[result.difficulty]?.title || result.difficulty} · Puzzle #${result.id}`;
  if (practiceFocus) {
    const solution = await api('/api/solve', puzzleString());
    states = solution.states;
    setUpChallenge();
  }
  message('');
  scheduleValidation();
}));
$('clear').addEventListener('click', () => {
  cancelValidation();
  resetEdits();
  original = Array(81).fill(0); givenClues = null; states = []; current = 0; puzzleLabel = ''; showIssues([]);
  entryMode = false; practiceFocus = ''; challengePending = false; practiceTarget = -1;
  transcribing = true; referencePhoto = ''; $('reference-image').removeAttribute('src');
  message(''); render(); $('random').focus();
  saveSession(true);
});
$('edit').addEventListener('click', () => { states = []; current = 0; challengePending = false; showIssues([]); message(''); render(); scheduleValidation(); });
function navigate(index) { challengePending = false; current = Math.max(0, Math.min(states.length - 1, index)); render(); }
$('reveal-hint').addEventListener('click', () => { if (!busy && practiceTarget > 0) navigate(practiceTarget); });
$('all-steps').addEventListener('click', () => navigate(0));
$('first').addEventListener('click', () => navigate(0));
$('previous').addEventListener('click', () => navigate(current - 1));
$('next').addEventListener('click', () => navigate(current + 1));
$('last').addEventListener('click', () => navigate(states.length - 1));
$('timeline').addEventListener('input', event => navigate(Number(event.target.value)));
$('candidates').addEventListener('change', render);
$('visuals').addEventListener('change', render);
$('proof-branch').addEventListener('change', event => { selectedProof = Number(event.target.value); render(); });
$('clear-cell').addEventListener('click', () => editCell(selectedCell, 0));
$('undo').addEventListener('click', undoEdit);
for (const button of $('number-pad').querySelectorAll('button')) {
  button.addEventListener('click', () => {
    const index = selectedCell;
    editCell(index, Number(button.dataset.digit));
    if (transcribing && canEditCell(index)) focusNextEmptyCell(index);
  });
}
keypadLayout.addEventListener('change', render);
new ResizeObserver(() => renderVisuals(states[current], states[current]?.step)).observe(board);
render();

function setUpChallenge() {
  if (!practiceFocus) return;
  practiceTarget = training.targetIndex(states, practiceFocus);
  if (practiceTarget < 1) {
    const focus = practiceFocus;
    practiceFocus = ''; states = []; current = 0; challengePending = false;
    throw Error(`This position no longer needs ${focus}. You can still view its steps or choose another puzzle.`);
  }
  current = practiceTarget - 1;
  challengePending = true;
  $('candidates').checked = true;
}

function renderChooser() {
  const level = training.levels[selectedDifficulty];
  for (const radio of document.querySelectorAll('[name="difficulty"]')) radio.checked = radio.value === selectedDifficulty;
  const catalog = practiceCatalog.find(level => level.difficulty === selectedDifficulty);
  const select = $('technique-focus'); select.replaceChildren();
  const mixed = document.createElement('option'); mixed.value = ''; mixed.textContent = 'A mix of techniques'; select.append(mixed);
  for (const currentLevel of [true, false]) {
    const techniques = (catalog?.techniques || []).filter(t => (t.difficulty === selectedDifficulty) === currentLevel);
    if (!techniques.length) continue;
    const group = document.createElement('optgroup'); group.label = currentLevel ? `${level.title} techniques` : 'Earlier techniques in these puzzles';
    for (const technique of techniques) {
      const option = document.createElement('option'); option.value = technique.name;
      option.textContent = `${technique.name} (${technique.count})`; group.append(option);
    }
    select.append(group);
  }
  if (catalog && ![...select.options].some(option => option.value === selectedTechnique)) selectedTechnique = '';
  select.value = selectedTechnique;
  $('focus-description').hidden = !selectedTechnique;
  $('focus-description').textContent = selectedTechnique ? training.describe(selectedTechnique) : '';
  renderTechniqueLink('focus-guide', selectedTechnique);
  const matching = selectedTechnique ? catalog?.techniques.find(t => t.name === selectedTechnique)?.count : catalog?.count;
  $('library-count').textContent = matching ? `${matching} ${selectedTechnique ? 'matching ' : ''}puzzles` : practiceCatalog.length ? 'No matching puzzles' : 'Loading practice library…';
  $('random').textContent = selectedTechnique ? 'Start technique practice' : 'Start puzzle';
  select.disabled = busy || !catalog;
  updateControls();
}

for (const radio of document.querySelectorAll('[name="difficulty"]')) radio.addEventListener('change', event => {
  selectedDifficulty = event.target.value; selectedTechnique = ''; renderChooser();
});
$('technique-focus').addEventListener('change', event => { selectedTechnique = event.target.value; renderChooser(); });
$('enter-puzzle').addEventListener('click', () => {
  if (busy) return;
  entryMode = true; render(); cells[0].input.focus();
});
renderChooser();
(async () => {
  try { practiceCatalog = await api('/api/puzzles/catalog'); renderChooser(); }
  catch {
    $('library-count').textContent = 'Practice library unavailable';
    $('chooser-error').textContent = 'You can still enter your own puzzle or import a photo.'; $('chooser-error').hidden = false;
  }
})();

// Placements explain existing restrictions. Only explicit elimination steps
// remove candidates; the UI never infers peer cleanup from a placement diff.
function visualData(state, step) {
  const visual = step?.visual || {};
  const proof = visual.proofs?.[selectedProof];
  const inferences = proof ? proof.nodes.flatMap(node => (node.parents || []).map(p => ({
    from: proof.nodes[p], to: node, kind: 'implication', reason: node.reason
  }))) : visual.inferences || [];
  const sources = proof ? [...new Map(proof.nodes.map(c => [`${c.row},${c.col}`, c])).values()] : visual.sources || [];
  return {
    sources, digits: visual.digits || [], units: visual.units || [],
    links: visual.links || [], candidateLinks: visual.candidateLinks || [],
    highlights: proof ? proof.nodes.map(c => ({ ...c, role: c.true ? 'color-a' : 'color-b' })) : visual.highlights || [], inferences, exact: !!visual.exact,
    restrictions: visual.restrictions || [], targets: step?.eliminations || []
  };
}
function renderVisuals(state, step) {
  if (challengePending) step = null;
  const overlay = $('deduction-overlay');
  overlay.replaceChildren();
  for (const { cell, marks } of cells) {
    cell.classList.remove('focus-unit', 'source-cell', 'target-cell', 'fin-cell');
    for (const mark of marks.children) mark.classList.remove('evidence', 'fin', 'color-a', 'color-b');
    // Remove display-only restriction marks before redrawing guides.
    for (const mark of marks.querySelectorAll('[data-propagated]')) { mark.textContent = ''; mark.classList.remove('removed', 'blocked'); delete mark.dataset.propagated; }
  }
  $('visual-summary').hidden = true;
  if (!state || !step || !$('visuals').checked) return;
  const { sources, digits, units, links, candidateLinks, highlights, inferences, exact, restrictions, targets } = visualData(state, step);
  const bounds = board.getBoundingClientRect();
  overlay.setAttribute('viewBox', `0 0 ${bounds.width} ${bounds.height}`);
  const svg = (tag, attributes) => {
    const element = document.createElementNS('http://www.w3.org/2000/svg', tag);
    for (const [key, value] of Object.entries(attributes)) element.setAttribute(key, value);
    overlay.append(element); return element;
  };
  const defs = document.createElementNS('http://www.w3.org/2000/svg', 'defs');
  const arrow = document.createElementNS('http://www.w3.org/2000/svg', 'marker');
  for (const [key, value] of Object.entries({ id: 'blocking-arrow', viewBox: '0 0 8 8', refX: 7, refY: 4, markerWidth: 5, markerHeight: 5, orient: 'auto-start-reverse' })) arrow.setAttribute(key, value);
  const arrowShape = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  arrowShape.setAttribute('d', 'M0,1 L7,4 L0,7'); arrowShape.setAttribute('fill', 'none'); arrowShape.setAttribute('stroke', '#ab694d'); arrowShape.setAttribute('stroke-width', '1.5');
  arrow.append(arrowShape); defs.append(arrow); overlay.append(defs);
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
    const sameCell = p => p.row === source.row && p.col === source.col;
    const linkedDigits = [...new Set([
      ...candidateLinks.filter(link => sameCell(link.from) || sameCell(link.to)).map(link => link.number),
      ...inferences.flatMap(link => [link.from, link.to].filter(sameCell).map(c => c.number))
    ])];
    for (const digit of linkedDigits.length ? linkedDigits : digits) {
      const mark = entry.marks.querySelector(`[data-digit="${digit}"]`);
      if (mark?.textContent && !mark.classList.contains('removed')) mark.classList.add('evidence');
    }
  }
  for (const highlight of highlights) {
    if (!['fin', 'color-a', 'color-b'].includes(highlight.role)) continue;
    const entry = cellAt(highlight);
    if (highlight.role === 'fin') entry.cell.classList.add('fin-cell');
    const mark = entry.marks.querySelector(`[data-digit="${highlight.number}"]`);
    if (mark?.textContent && !mark.classList.contains('removed')) mark.classList.add(highlight.role);
  }
  // Locked candidates get one clear axis through their source cells.
  if (!exact && !step.number && !links.length && !candidateLinks.length && sources.length > 1 && digits.length === 1) {
    const points = sources.map(source => point(source, digits[0]));
    if (sources.every(p => p.row === sources[0].row) || sources.every(p => p.col === sources[0].col)) {
      const a = points[0], b = points.at(-1);
      svg('line', { x1: a.x, y1: a.y, x2: b.x, y2: b.y, class: 'guide-axis' });
    }
  }
  const markTarget = (target, restricted) => {
    const entry = cellAt(target);
    entry.cell.classList.add('target-cell');
    const mark = entry.marks.querySelector(`[data-digit="${target.number}"]`);
    if (mark && !mark.textContent) {
      mark.textContent = target.number;
      mark.classList.add(restricted ? 'blocked' : 'removed');
      mark.dataset.propagated = 'true';
    }
    // A placement cell shows its placed value; do not circle that value as if
    // it were the candidate being ruled out.
    if (state.board[target.row][target.col]) return;
    const end = point(target, target.number);
    const radius = $('candidates').checked ? Math.max(5, bounds.width / 70) : bounds.width / 34;
    svg('circle', { cx: end.x, cy: end.y, r: radius, class: restricted ? 'guide-restriction' : 'guide-ring' });
    if (!$('candidates').checked) svg('path', { d: `M${end.x - 4},${end.y - 4}l8,8m0,-8l-8,8`, class: 'guide-cross' });
  };
  const connect = (source, target, number, actualBlocker, toNumber = number) => {
    const start = point(source, number), end = point(target, toNumber);
    if (state.board[target.row][target.col]) {
      // Terminate at the edge of the destination cell, leaving its digit clear.
      const distance = Math.hypot(end.x - start.x, end.y - start.y);
      const inset = bounds.width / 9 * .34;
      if (distance > inset) { end.x -= (end.x - start.x) / distance * inset; end.y -= (end.y - start.y) / distance * inset; }
    }
    const line = svg('line', {
      x1: start.x, y1: start.y, x2: end.x, y2: end.y,
      class: actualBlocker ? 'guide-link guide-blocker' : 'guide-link',
      'data-from': `${source.row},${source.col}`, 'data-to': `${target.row},${target.col}`, 'data-number': number,
      ...(actualBlocker ? { 'marker-end': 'url(#blocking-arrow)' } : {})
    });
    const title = document.createElementNS('http://www.w3.org/2000/svg', 'title');
    title.textContent = `R${source.row + 1}C${source.col + 1} ${actualBlocker ? 'contains' : 'reserves'} ${number}: ${number} cannot go in R${target.row + 1}C${target.col + 1}.`;
    line.append(title);
    return line;
  };
  for (const restriction of restrictions) markTarget(restriction, true);
  // Exact source-to-cell relationships supplied by Go take precedence.
  for (const link of links) connect(link.from, link.to, link.number, true);
  for (const link of candidateLinks) {
    const line = connect(link.from, link.to, link.number, false);
    line.dataset.kind = link.kind;
    line.classList.add(`guide-${link.kind}`);
    const from = `R${link.from.row + 1}C${link.from.col + 1}`, to = `R${link.to.row + 1}C${link.to.col + 1}`;
    line.querySelector('title').textContent = link.kind === 'strong'
      ? `${link.number} must go in one of ${from} or ${to} in ${link.unit.kind} ${link.unit.index + 1}.`
      : link.kind === 'weak'
        ? `${from} and ${to} cannot both contain ${link.number}.`
        : `The highlighted pattern rules out ${link.number} in ${to}; ${from} is a supporting candidate.`;
  }
  for (const link of inferences) {
    const line = connect(link.from, link.to, link.from.number, false, link.to.number);
    line.dataset.kind = link.kind;
    line.dataset.toNumber = link.to.number;
    line.classList.add(`guide-${link.kind}`);
    line.querySelector('title').textContent = link.reason;
  }
  for (const target of targets) {
    markTarget(target, false);
    if (exact || links.length || candidateLinks.length || inferences.length) continue;
    const sameCell = sources.some(source => source.row === target.row && source.col === target.col);
    if (!sameCell && sources.length) {
      // Locked-candidate evidence contains unplaced candidates; these are
      // deduction links rather than claims about an already placed digit.
      const end = point(target, target.number);
      // For mixed triples/quads, only a cell that actually allows this digit
      // may anchor the deduction line.
      const supporting = sources.filter(source =>
        state.board[source.row][source.col] === target.number ||
        (state.candidates[source.row][source.col] & (1 << target.number)) !== 0);
      if (!supporting.length) continue;
      const source = supporting.reduce((a, b) => {
        const p = point(a, target.number), q = point(b, target.number);
        return Math.hypot(p.x - end.x, p.y - end.y) < Math.hypot(q.x - end.x, q.y - end.y) ? a : b;
      });
      connect(source, target, target.number, false);
    }
  }
  const unitNames = units.map(u => `${u.kind === 'column' ? 'Column' : u.kind === 'box' ? 'Box' : 'Row'} ${u.index + 1}`);
  const text = [];
  if (unitNames.length) text.push(unitNames.join(' + '));
  if (restrictions.length) text.push(`${restrictions.length} alternative${restrictions.length === 1 ? '' : 's'} ruled out`);
  if (targets.length) text.push(`${targets.length} candidate${targets.length === 1 ? '' : 's'} removed`);
  if (highlights.some(h => h.role === 'fin')) text.push('Amber: fins');
  if (highlights.some(h => h.role === 'color-a')) text.push('Blue / purple: opposite assignments');
  $('visual-summary').textContent = text.join(' · ');
  $('visual-summary').hidden = !text.length;
}

// Photo review finalizes the locked clues before loading the main board.
document.addEventListener('godoku:photo-import', event => {
  if (busy) return;
  resetEdits();
  original = [...event.detail.values];
  givenClues = [...original];
  transcribing = false; referencePhoto = event.detail.image || '';
  if (referencePhoto) $('reference-image').src = referencePhoto;
  else $('reference-image').removeAttribute('src');
  states = []; current = 0; puzzleLabel = 'Puzzle from your photo';
  entryMode = true; practiceFocus = ''; challengePending = false;
  showIssues([]);
  message('');
  render();
  scheduleValidation();
  cells[Math.max(0, original.findIndex(value => !value))].input.focus();
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

// Recompute history from the saved submission rather than storing candidate
// snapshots. The clue colors and editing mode remain independent of playback.
async function restorePlayback() {
  message('Restoring solving steps…');
  try {
    const result = await api('/api/solve', puzzleString());
    states = result.states;
    current = Math.max(0, Math.min(restoringStep, states.length - 1));
    practiceTarget = practiceFocus ? training.targetIndex(states, practiceFocus) : -1;
    if (practiceTarget < 1 || current !== practiceTarget - 1) challengePending = false;
    if (givenClues === null) givenClues = [...original];
    message(result.solved ? '' : 'Stopped: no more supported moves.');
  } catch {
    message('Puzzle restored. The solving steps could not be loaded; try Solve again.', 'error');
  } finally {
    restoringStep = null; busy = false; render();
  }
}
if (restoringStep !== null) restorePlayback();
else if (entryMode) scheduleValidation();
