'use strict';
// Only the static build selects WASM; the existing Go-server UI keeps its API.
if (document.documentElement.dataset.engine === 'wasm') {
  let worker, ready, nextID = 0;
  const pending = new Map();
  function fail(error) {
    worker?.terminate(); worker = null; ready = null;
    for (const { reject, timer } of pending.values()) { clearTimeout(timer); reject(error); }
    pending.clear();
  }
  function start() {
    if (ready) return ready;
    ready = new Promise((resolve, reject) => {
      const timer = setTimeout(() => { const error = Error('The browser solver took too long to load. Check your connection and retry.'); reject(error); fail(error); }, 60000);
      worker = new Worker(new URL('solver-worker.js', document.baseURI));
      worker.onmessage = ({ data }) => {
        if (data.type === 'ready') { clearTimeout(timer); resolve(); return; }
        if (data.type === 'fatal') { clearTimeout(timer); const error = Error(data.error); reject(error); fail(error); return; }
        const request = pending.get(data.id);
        if (!request) return;
        pending.delete(data.id); clearTimeout(request.timer);
        if (data.error) request.reject(Error(data.error)); else request.resolve(data.response);
      };
      worker.onerror = () => { clearTimeout(timer); const error = Error('Could not start the browser solver. Reload the page or try another browser.'); reject(error); fail(error); };
    });
    return ready;
  }
  window.godokuEngine = {
    async request(path, puzzle = '') {
      await start();
      return new Promise((resolve, reject) => {
        const id = ++nextID;
        const timer = setTimeout(() => fail(Error('The browser solver timed out. Please try again.')), 60000);
        pending.set(id, { resolve, reject, timer });
        worker.postMessage({ id, path, puzzle });
      });
    }
  };
}
