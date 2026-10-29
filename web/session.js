'use strict';

// One puzzle per browser. Store the submission and replay position; the solver
// rebuilds its history on return so old candidate data cannot undo deductions.
window.godokuSession = (() => {
  const key = 'godoku.session.v1';
  let lastSaved = null;
  const validValues = values => Array.isArray(values) && values.length === 81 &&
    values.every(value => Number.isInteger(value) && value >= 0 && value <= 9);

  function load() {
    try {
      const text = localStorage.getItem(key);
      const saved = JSON.parse(text);
      if (!saved || saved.version !== 1 || !validValues(saved.original) ||
          !(saved.givenClues === null || validValues(saved.givenClues)) ||
          typeof saved.transcribing !== 'boolean') return null;
      lastSaved = text;
      const reviewing = saved.reviewing === true;
      return {
        original: saved.original, givenClues: saved.givenClues,
        transcribing: saved.transcribing && !reviewing,
        puzzleLabel: typeof saved.puzzleLabel === 'string' ? saved.puzzleLabel : '',
        referencePhoto: typeof saved.referencePhoto === 'string' && /^data:image\/(jpeg|png|webp);base64,/.test(saved.referencePhoto) ? saved.referencePhoto : '',
        reviewing,
        current: Number.isSafeInteger(saved.current) && saved.current >= 0 ? saved.current : 0,
        candidates: saved.candidates !== false, guides: saved.guides !== false
      };
    } catch { return null; }
  }

  // Return a result so storage failures never interrupt editing or solving.
  function save(snapshot) {
    try {
      if (!snapshot) {
        localStorage.removeItem(key); lastSaved = null;
        return 'saved';
      }
      const text = JSON.stringify({ version: 1, ...snapshot });
      if (text !== lastSaved) localStorage.setItem(key, text);
      lastSaved = text;
      return 'saved';
    } catch {
      // If a photo fills the quota, retain the numbers and resume position.
      if (snapshot?.referencePhoto) {
        try {
          const text = JSON.stringify({ version: 1, ...snapshot, referencePhoto: '' });
          if (text !== lastSaved) localStorage.setItem(key, text);
          lastSaved = text;
          return 'without-photo';
        } catch { /* The browser may block storage entirely. */ }
      }
      return 'failed';
    }
  }
  return { load, save };
})();
