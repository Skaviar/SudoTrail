# Build and deployment

[Back to the README](../README.md).

## Static Go WebAssembly build (Cloudflare Pages)

The website can run entirely in the browser: the Go solver compiles to
WebAssembly and runs in a Web Worker, with 2,000 difficulty-rated practice
puzzles embedded inside that WASM binary. Cloudflare only serves files; it does
not execute Go or receive puzzle submissions. The full CSV and its solutions
are not shipped. Building the static site does not require the large local CSV.

Build from the project root with Go 1.25.6 or newer:

```sh
go run ./cmd/buildweb
```

This writes `dist/`: HTML/CSS/JavaScript, `solver.wasm`, and the matching Go
`wasm_exec.js`. Generated output is ignored by Git. Rebuild after source changes.
The Go WASM runtime support file must match the Go version used to compile WASM;
the build command copies it from the same toolchain automatically.

Preview the exact static output:

```sh
go run ./cmd/web -static dist
```

Open `http://127.0.0.1:8080`. This mode exposes no API and does not load the CSV.
Use localhost for local WASM testing or HTTPS when hosted; the Go runtime needs
browser cryptography APIs that may be unavailable on insecure LAN HTTP origins.

## Cloudflare Pages

For a project connected to Git, set:

- **Build command:** `make build`
- **Build output directory:** `dist`

For Direct Upload:

1. Run `go run ./cmd/buildweb` locally.
2. In Cloudflare's dashboard, create a **Pages** project using **Direct Upload**.
3. Upload the **dist folder** and deploy. Do not upload the project or the CSV.
4. Open the HTTPS `pages.dev` address provided by Cloudflare.

Alternatively, with Wrangler installed and authenticated:

```sh
npx wrangler pages deploy dist --project-name sudotrail
```

This is a static Pages deployment, not a Cloudflare Worker running Go. It fits
Cloudflare's free static hosting model; the build checks its WASM file against
the 25 MiB per-asset limit. See [Direct Upload](https://developers.cloudflare.com/pages/get-started/direct-upload/)
and [Pages limits](https://developers.cloudflare.com/pages/platform/limits/).
A Direct Upload Pages project cannot later switch to Git integration in-place.

Normal solving, validation, random selection, photo processing, and history
navigation run in the visitor's browser. OCR still downloads its engine and
language data from public CDNs. No service worker/offline installation is added;
the site must initially load over HTTP(S).

The command entry points are:

| Command | Mode |
| --- | --- |
| `go run . -benchmark` | Full-database benchmark |
| `go run . -rate` | Difficulty export |
| `go run ./cmd/web` | Embedded UI with the local Go HTTP API; optional CSV for unfiltered random selection |
| `go run ./cmd/buildweb` | Generate a deployable static WASM website |
| `go run ./cmd/web -static dist` | Preview static WASM output |

The HTTP API and WASM entry point share `internal/puzzleapp`, so snapshots,
visual guides, errors, and solved/stuck behavior use the same solver logic.
`internal/puzzlebank` owns the balanced practice catalog; `cmd/wasm` bridges Go to
JavaScript. `web/engine.js` and `web/solver-worker.js` load and call WASM without
blocking board input or rendering. The generated HTML selects WASM mode; the
server-embedded HTML selects API mode.
