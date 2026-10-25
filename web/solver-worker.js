'use strict';
// Keep Go computation off the UI thread, especially on mobile devices.
self.onmessage = ({ data }) => {
  try { self.postMessage({ id: data.id, response: JSON.parse(self.godokuRequest(data.path, data.puzzle)) }); }
  catch { self.postMessage({ id: data.id, error: 'The browser solver could not process this puzzle.' }); }
};
(async () => {
  try {
    importScripts('wasm_exec.js');
    const go = new Go();
    self.godokuReady = () => self.postMessage({ type: 'ready' });
    const response = await fetch('solver.wasm');
    if (!response.ok) throw Error(`Solver download failed (${response.status}).`);
    // ArrayBuffer supports static hosts even if their WASM MIME type is missing.
    const { instance } = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
    go.run(instance).catch(error => self.postMessage({ type: 'fatal', error: error.message }));
  } catch (error) { self.postMessage({ type: 'fatal', error: error.message || 'Could not load the browser solver.' }); }
})();
