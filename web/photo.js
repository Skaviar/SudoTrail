'use strict';

// Browser-only OCR: the image is never sent to the Go server or the CDN.
(() => {
  const $ = id => document.getElementById(id);
  const dialog = $('photo-dialog');
  const canvas = $('photo-canvas');
  const context = canvas.getContext('2d');
  let image = null, corners = [], drag = -1, running = false, loading = false, revision = 0;
  let worker = null, values = Array(81).fill(0);
  const review = [];
  let enginePromise;

  function status(text, kind = '') { $('photo-status').textContent = text; $('photo-status').className = `message ${kind}`; }
  function controls() {
    $('photo-read').disabled = !image || running || loading;
    $('photo-use').disabled = running || loading || !values.some(Boolean);
    $('photo-adjust').disabled = running || loading;
    $('photo-rotate').disabled = running || loading;
    $('photo-file').disabled = $('photo-camera').disabled = running || loading;
    for (const { input } of review) input.readOnly = running || loading;
  }
  function setReviewing(ready) {
    dialog.classList.toggle('has-review', ready);
    $('photo-review').hidden = $('photo-use').hidden = $('photo-preview').hidden = $('photo-adjust').hidden = !ready;
    $('photo-title').textContent = ready ? 'Check the clues' : 'Import a Sudoku photo';
  }
  function invalidate() { revision++; setReviewing(false); }
  for (let i = 0; i < 81; i++) {
    const cell = document.createElement('div'); cell.className = 'cell';
    const input = document.createElement('input'); input.type = 'text'; input.inputMode = 'numeric'; input.autocomplete = 'off';
    input.setAttribute('aria-label', `Detected row ${Math.floor(i / 9) + 1}, column ${i % 9 + 1}`);
    input.addEventListener('focus', () => input.select());
    input.addEventListener('input', event => {
      if (running || loading || event.isComposing) return;
      const enteredDigit = /[1-9]/.test(event.data ?? input.value);
      const wasFocused = document.activeElement === input;
      const insertedText = event.inputType?.startsWith('insert') || event.data != null;
      if (wasFocused && insertedText && !enteredDigit) {
        input.value = values[i] || '';
        focusNextEmptyCell(i, values, review);
        return;
      }
      const digit = input.value.replace(/[^1-9]/g, '').at(-1);
      values[i] = digit ? Number(digit) : 0; input.value = digit || '';
      cell.classList.remove('uncertain'); reviewConflicts();
      if (digit && enteredDigit && wasFocused) focusNextEmptyCell(i, values, review);
    });
    input.addEventListener('keydown', event => handleGridKey(event, i, review, values, !running && !loading));
    cell.append(input); $('photo-review-grid').append(cell); review.push({ cell, input });
  }
  function reviewConflicts() {
    const conflicts = new Set();
    for (let kind = 0; kind < 3; kind++) for (let unit = 0; unit < 9; unit++) {
      const seen = new Map();
      for (let p = 0; p < 9; p++) {
        const row = kind === 0 ? unit : kind === 1 ? p : Math.floor(unit / 3) * 3 + Math.floor(p / 3);
        const col = kind === 0 ? p : kind === 1 ? unit : unit % 3 * 3 + p % 3;
        const i = row * 9 + col, value = values[i];
        if (!value) continue;
        if (seen.has(value)) { conflicts.add(i); conflicts.add(seen.get(value)); } else seen.set(value, i);
      }
    }
    review.forEach(({ cell, input }, i) => { cell.classList.toggle('conflict', conflicts.has(i)); input.setAttribute('aria-invalid', String(conflicts.has(i))); });
    const count = values.filter(Boolean).length;
    $('photo-review-status').textContent = `${count} clues · ${conflicts.size ? `${conflicts.size} conflicting cells` : 'No conflicts'}`;
    $('photo-use').disabled = count === 0;
  }
  function draw() {
    if (!image) return;
    context.drawImage(image, 0, 0, canvas.width, canvas.height);
    context.beginPath(); corners.forEach((p, i) => i ? context.lineTo(p.x, p.y) : context.moveTo(p.x, p.y)); context.closePath();
    context.fillStyle = '#b3db9a30'; context.fill(); context.strokeStyle = '#e8ff7f'; context.lineWidth = Math.max(2, canvas.width / 250); context.stroke();
    // Perspective-aligned guide lines make all nine cell boundaries visible.
    try {
      const transform = homography(corners);
      context.strokeStyle = '#e8ff7f80'; context.lineWidth = 1;
      for (let k = 1; k < 9; k++) for (const vertical of [true, false]) {
        const a = project(transform, vertical ? k / 9 : 0, vertical ? 0 : k / 9);
        const b = project(transform, vertical ? k / 9 : 1, vertical ? 1 : k / 9);
        context.beginPath(); context.moveTo(a.x, a.y); context.lineTo(b.x, b.y); context.stroke();
      }
    } catch { /* Invalid corner positions are reported before recognition. */ }
    corners.forEach((p, i) => {
      const radius = Math.max(11, canvas.width / 38);
      context.beginPath(); context.arc(p.x, p.y, radius, 0, Math.PI * 2); context.fillStyle = '#245b49'; context.fill(); context.strokeStyle = 'white'; context.lineWidth = 2; context.stroke();
      context.fillStyle = 'white'; context.font = `bold ${radius}px sans-serif`; context.textAlign = 'center'; context.textBaseline = 'middle'; context.fillText(i + 1, p.x, p.y);
    });
  }
  function pointer(event) {
    const bounds = canvas.getBoundingClientRect();
    return { x: Math.max(0, Math.min(canvas.width, (event.clientX - bounds.left) * canvas.width / bounds.width)), y: Math.max(0, Math.min(canvas.height, (event.clientY - bounds.top) * canvas.height / bounds.height)) };
  }
  canvas.addEventListener('pointerdown', event => {
    if (running || loading || !image) return;
    const p = pointer(event);
    drag = corners.reduce((best, corner, i) => Math.hypot(corner.x - p.x, corner.y - p.y) < Math.hypot(corners[best].x - p.x, corners[best].y - p.y) ? i : best, 0);
    canvas.setPointerCapture(event.pointerId); corners[drag] = p; invalidate(); draw();
  });
  canvas.addEventListener('pointermove', event => { if (drag < 0 || running) return; corners[drag] = pointer(event); draw(); });
  for (const type of ['pointerup', 'pointercancel']) canvas.addEventListener(type, () => { drag = -1; });

  async function load(file) {
    if (!file) return;
    if (file.size > 20 * 1024 * 1024) { status('Choose a photo smaller than 20 MB.', 'error'); return; }
    const token = ++revision;
    loading = true; controls(); status('Opening the photo…');
    let bitmap;
    try {
      bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' });
      if (token !== revision) return;
      const scale = Math.min(1, 1400 / Math.max(bitmap.width, bitmap.height));
      const source = document.createElement('canvas'); source.width = Math.round(bitmap.width * scale); source.height = Math.round(bitmap.height * scale);
      source.getContext('2d').drawImage(bitmap, 0, 0, source.width, source.height); image = source;
      canvas.width = source.width; canvas.height = source.height;
      corners = [{ x: 0, y: 0 }, { x: canvas.width, y: 0 }, { x: canvas.width, y: canvas.height }, { x: 0, y: canvas.height }];
      invalidate(); draw(); controls(); $('photo-editor').hidden = false;
      status('Align all four corners with the outer grid border, then read the grid.');
    } catch { status('This photo could not be opened. Try a JPEG, PNG, or WebP image.', 'error'); }
    finally { bitmap?.close(); loading = false; controls(); }
  }
  for (const id of ['photo-file', 'photo-camera']) {
    const label = document.querySelector(`label[for="${id}"]`);
    label.tabIndex = 0; label.setAttribute('role', 'button');
    label.addEventListener('keydown', event => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); if (!$(id).disabled) $(id).click(); } });
  }
  for (const id of ['photo-file', 'photo-camera']) $(id).addEventListener('change', event => { load(event.target.files[0]); event.target.value = ''; });
  $('photo-rotate').addEventListener('click', () => {
    if (!image || running || loading) return;
    const rotated = document.createElement('canvas'); rotated.width = image.height; rotated.height = image.width;
    const ctx = rotated.getContext('2d'); ctx.translate(rotated.width, 0); ctx.rotate(Math.PI / 2); ctx.drawImage(image, 0, 0);
    corners = [corners[3], corners[0], corners[1], corners[2]].map(p => ({ x: image.height - p.y, y: p.x }));
    image = rotated; canvas.width = image.width; canvas.height = image.height; invalidate(); draw();
  });
  $('photo-adjust').addEventListener('click', () => {
    if (running || loading) return;
    invalidate(); draw();
    status('Adjust the corners, then read the grid again.');
  });

  // Map a unit square onto the selected quadrilateral by solving eight equations.
  function homography(points) {
    const matrix = [], coordinates = [[0, 0], [1, 0], [1, 1], [0, 1]];
    coordinates.forEach(([u, v], i) => {
      const { x, y } = points[i];
      matrix.push([u, v, 1, 0, 0, 0, -x * u, -x * v, x]);
      matrix.push([0, 0, 0, u, v, 1, -y * u, -y * v, y]);
    });
    for (let col = 0; col < 8; col++) {
      let pivot = col;
      for (let row = col + 1; row < 8; row++) if (Math.abs(matrix[row][col]) > Math.abs(matrix[pivot][col])) pivot = row;
      [matrix[col], matrix[pivot]] = [matrix[pivot], matrix[col]];
      const divisor = matrix[col][col]; if (Math.abs(divisor) < 1e-8) throw Error('Adjust the corners to form a complete grid.');
      for (let k = col; k <= 8; k++) matrix[col][k] /= divisor;
      for (let row = 0; row < 8; row++) if (row !== col) { const factor = matrix[row][col]; for (let k = col; k <= 8; k++) matrix[row][k] -= factor * matrix[col][k]; }
    }
    return matrix.map(row => row[8]);
  }
  function project(h, u, v) { const z = h[6] * u + h[7] * v + 1; return { x: (h[0] * u + h[1] * v + h[2]) / z, y: (h[3] * u + h[4] * v + h[5]) / z }; }
  function straighten() {
    // Reject crossed, reversed, tiny, or nearly collinear corner selections.
    for (let i = 0; i < 4; i++) {
      const a = corners[i], b = corners[(i + 1) % 4], c = corners[(i + 2) % 4];
      if ((b.x - a.x) * (c.y - b.y) - (b.y - a.y) * (c.x - b.x) < 100) throw Error('Place handles 1–4 clockwise around all four outer corners.');
    }
    const h = homography(corners), source = image.getContext('2d').getImageData(0, 0, image.width, image.height);
    const output = document.createElement('canvas'); output.width = output.height = 900;
    const ctx = output.getContext('2d'), pixels = ctx.createImageData(900, 900);
    for (let y = 0; y < 900; y++) for (let x = 0; x < 900; x++) {
      const p = project(h, (x + .5) / 900, (y + .5) / 900);
      const sx = Math.max(0, Math.min(image.width - 1, Math.round(p.x))), sy = Math.max(0, Math.min(image.height - 1, Math.round(p.y)));
      const from = (sy * image.width + sx) * 4, to = (y * 900 + x) * 4;
      pixels.data[to] = source.data[from]; pixels.data[to + 1] = source.data[from + 1]; pixels.data[to + 2] = source.data[from + 2]; pixels.data[to + 3] = 255;
    }
    ctx.putImageData(pixels, 0, 0); return output;
  }
  function digitImage(grid, index) {
    // Trim grid lines, then retain digit-sized connected components.
    const size = 80, pixels = grid.getContext('2d').getImageData(index % 9 * 100 + 10, Math.floor(index / 9) * 100 + 10, size, size);
    const gray = new Uint8Array(size * size), histogram = new Uint32Array(256);
    let sum = 0;
    for (let i = 0; i < gray.length; i++) { const offset = i * 4; gray[i] = Math.round(.299 * pixels.data[offset] + .587 * pixels.data[offset + 1] + .114 * pixels.data[offset + 2]); histogram[gray[i]]++; sum += gray[i]; }
    // Otsu threshold adapts to paper shadows instead of assuming pure white.
    let left = 0, leftSum = 0, variance = 0, threshold = 128;
    for (let t = 0; t < 255; t++) {
      left += histogram[t]; leftSum += t * histogram[t]; const right = gray.length - left;
      if (!left || !right) continue;
      const delta = leftSum / left - (sum - leftSum) / right, score = left * right * delta * delta;
      if (score > variance) { variance = score; threshold = t; }
    }
    const seen = new Uint8Array(gray.length), components = [];
    for (let i = 0; i < gray.length; i++) {
      if (seen[i] || gray[i] > threshold || gray[i] > 200) continue;
      const queue = [i], points = []; seen[i] = 1;
      let minX = size, maxX = 0, minY = size, maxY = 0;
      for (let q = 0; q < queue.length; q++) {
        const p = queue[q], x = p % size, y = Math.floor(p / size); points.push(p);
        minX = Math.min(minX, x); maxX = Math.max(maxX, x); minY = Math.min(minY, y); maxY = Math.max(maxY, y);
        for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) {
          const nx = x + dx, ny = y + dy, n = ny * size + nx;
          if (nx >= 0 && nx < size && ny >= 0 && ny < size && !seen[n] && gray[n] <= threshold && gray[n] <= 200) { seen[n] = 1; queue.push(n); }
        }
      }
      const width = maxX - minX + 1, height = maxY - minY + 1;
      if (points.length >= 28 && height >= 18 && height <= 75 && width >= 3 && width <= 65 && minY > 0 && maxY < size - 1) components.push({ points, minX, maxX, minY, maxY });
    }
    components.sort((a, b) => b.points.length - a.points.length);
    if (!components.length) return null;
    const component = components[0], crop = document.createElement('canvas'); crop.width = crop.height = size;
    const ctx = crop.getContext('2d'); ctx.fillStyle = 'white'; ctx.fillRect(0, 0, size, size); ctx.fillStyle = 'black';
    for (const p of component.points) ctx.fillRect(p % size, Math.floor(p / size), 1, 1);
    const output = document.createElement('canvas'); output.width = output.height = 120;
    const dest = output.getContext('2d'); dest.fillStyle = 'white'; dest.fillRect(0, 0, 120, 120);
    const width = component.maxX - component.minX + 1, height = component.maxY - component.minY + 1, scale = 76 / Math.max(width, height);
    dest.drawImage(crop, component.minX, component.minY, width, height, (120 - width * scale) / 2, (120 - height * scale) / 2, width * scale, height * scale);
    return output;
  }
  function engine() {
    if (window.Tesseract) return Promise.resolve(window.Tesseract);
    if (!enginePromise) enginePromise = new Promise((resolve, reject) => {
      const script = document.createElement('script'); script.src = 'https://cdn.jsdelivr.net/npm/tesseract.js@6.0.1/dist/tesseract.min.js';
      const timeout = setTimeout(() => { script.remove(); enginePromise = null; reject(Error('OCR download timed out. Check your internet connection and try again.')); }, 30000);
      script.onload = () => { clearTimeout(timeout); resolve(window.Tesseract); };
      script.onerror = () => { clearTimeout(timeout); script.remove(); enginePromise = null; reject(Error('OCR could not download. Check your internet connection and try again.')); };
      document.head.append(script);
    });
    return enginePromise;
  }
  $('photo-read').addEventListener('click', async () => {
    if (!image || running || loading) return;
    invalidate(); const token = revision; running = true; controls();
    $('photo-progress').hidden = false; $('photo-progress').value = 0;
    let active;
    try {
      status('Preparing the grid and downloading the OCR engine on first use…');
      const grid = straighten(), Tesseract = await engine();
      if (token !== revision) return;
      active = await Tesseract.createWorker('eng', 1, { logger: event => { if (token === revision && event.status !== 'recognizing text') status(`OCR: ${event.status}…`); } });
      if (token !== revision) return;
      worker = active;
      await active.setParameters({ tessedit_char_whitelist: '123456789', tessedit_pageseg_mode: '6', user_defined_dpi: '300' });
      const detected = Array(81).fill(0), uncertain = new Set();
      for (let i = 0; i < 81; i++) {
        if (token !== revision) return;
        status(`Reading row ${Math.floor(i / 9) + 1} of 9…`);
        const digit = digitImage(grid, i);
        if (digit) {
          const { data } = await active.recognize(digit);
          const text = data.text.trim();
          if (/^[1-9]$/.test(text)) detected[i] = Number(text);
          if (!detected[i] || data.confidence < 75) uncertain.add(i);
        }
        $('photo-progress').value = i + 1;
        await new Promise(resolve => setTimeout(resolve, 0));
      }
      if (token !== revision) return;
      values = detected;
      review.forEach(({ cell, input }, i) => { input.value = values[i] || ''; cell.classList.toggle('uncertain', uncertain.has(i)); });
      $('photo-preview').src = grid.toDataURL('image/jpeg', .92);
      setReviewing(true);
      reviewConflicts();
      status(values.some(Boolean) ? 'Ready for review. Recognition can make mistakes; compare all clues before importing.' : 'No digits were recognized. Try a clearer photo or adjust the grid corners.', values.some(Boolean) ? '' : 'error');
      dialog.scrollTop = 0;
    } catch (error) { if (token === revision) status(error.message || 'Could not read this photo. Please try another image.', 'error'); }
    finally {
      if (active) { try { await active.terminate(); } catch {} }
      if (worker === active) worker = null;
      if (token === revision) { running = false; controls(); $('photo-progress').hidden = true; if (!$('photo-review').hidden) reviewConflicts(); }
    }
  });
  $('import-photo').addEventListener('click', () => { dialog.showModal(); controls(); });
  $('photo-close').addEventListener('click', () => dialog.close());
  dialog.addEventListener('close', () => { revision++; running = false; if (worker) { worker.terminate().catch(() => {}); worker = null; } $('photo-progress').hidden = true; controls(); });
  $('photo-use').addEventListener('click', () => {
    if (running || !values.some(Boolean)) return;
    const detail = { values: [...values], image: $('photo-preview').src };
    dialog.close();
    document.dispatchEvent(new CustomEvent('godoku:photo-import', { detail }));
  });
})();
