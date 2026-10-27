# Godoku

A Go Sudoku solver that explains one logical move at a time.

Run the terminal app with `go run .`. Keep `sudoku-3m.csv` in the project
root for database puzzles. Use `go test ./...` to run the tests.

## Quick commands

Run `make` to list the available commands. Go remains the only build dependency;
`make` provides shortcuts for the existing commands.

| Command | What it does |
| --- | --- |
| `make dev` | Rebuild WASM and serve the static website at http://127.0.0.1:8080 |
| `make build` | Generate `dist/` for Cloudflare deployment |
| `make preview` | Same as `make dev`; always rebuilds before serving |
| `make web` | Run the local Go HTTP API and website |
| `make run` | Run the terminal app and its benchmark menu |
| `make check` | Run tests and `go vet` |
| `make race` | Run tests with the race detector |
| `make fmt` | Format Go source |

Press Ctrl+C to stop a running app. Restart `make dev` after source changes;
it does not watch files automatically.

Override settings when needed:

```sh
make dev ADDR=127.0.0.1:9090
make web ADDR=0.0.0.0:8080
make web DATABASE=/path/to/puzzles.csv
make build DIST=dist
```

For Cloudflare Git integration, the build command can be `make build`, with
output directory `dist`. The existing `go run ./cmd/buildweb` command also works.

## Static Go WebAssembly build (Cloudflare Pages)

The website can run entirely in the browser: the Go solver compiles to
WebAssembly and runs in a Web Worker, while 1,000 random puzzles are embedded
inside that WASM binary. Cloudflare only serves files; it does not execute Go
or receive puzzle submissions. The full CSV and its solutions are not shipped.
The checked-in catalog is a fixed sample from the first 10,000 database rows;
building the static site does not require the large local CSV.

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

Deploy without a server:

1. Run `go run ./cmd/buildweb` locally.
2. In Cloudflare's dashboard, create a **Pages** project using **Direct Upload**.
3. Upload the **dist folder** and deploy. Do not upload the project or the CSV.
4. Open the HTTPS `pages.dev` address provided by Cloudflare.

Alternatively, with Wrangler installed and authenticated:

```sh
npx wrangler pages deploy dist --project-name godoku
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

The original programs remain available:

| Command | Mode |
| --- | --- |
| `go run .` | Terminal app and full-database benchmarks |
| `go run ./cmd/web` | Embedded UI with the local Go HTTP API/full CSV |
| `go run ./cmd/buildweb` | Generate a deployable static WASM website |
| `go run ./cmd/web -static dist` | Preview static WASM output |

The HTTP API and WASM entry point share `internal/puzzleapp`, so snapshots,
visual guides, errors, and solved/stuck behavior use the same solver logic.
`internal/puzzlebank` owns the 1,000 embedded puzzles; `cmd/wasm` bridges Go to
JavaScript. `web/engine.js` and `web/solver-worker.js` load and call WASM without
blocking board input or rendering. The generated HTML selects WASM mode; the
server-embedded HTML selects API mode.

## Reading the code

Start with `solver/solve.go`: `SolveStep` tries techniques in order and
stops after the first successful change. Each `Apply...` function changes
the board or candidates **before** returning its explanation.

- `solver/types.go`: the puzzle input and Sudoku state.
- `solver/sudoku.go`: input validation, candidate initialization, placement,
  and checking whether the puzzle is solved.
- `solver/candidates.go`: candidate sets stored as bits; bit 5 means digit 5
  is possible. Bit 0 is unused.
- `solver/units.go`: a unit is a row, column, or box. Each has nine named
  cell positions. Coordinates are zero-based internally and one-based in text.
- `solver/step.go`: the placement or eliminations and their explanation.
- Technique files: explicit loops that find and apply one deduction.
  Explanation helpers live beside the technique that uses them.
- `solver/history.go`: an initial snapshot followed by the state after each
  step. Both the board and candidates are copied. A UI can display any entry
  by index without solving again.
- `main.go` and `cli.go`: database loading, menu, terminal rendering, and
  pausing between steps. The solver package has no terminal dependencies.

## Using the solver

```go
s, err := solver.NewSudoku(solver.Puzzle{Puzzle: puzzleString})
if err != nil {
    // Report the invalid input.
    return
}
history := s.Solve()
// history.Entries[0] is the original puzzle.
// Every later entry contains a Step and the resulting State.
// s.IsSolved() distinguishes success from stopping without a supported move.
_ = history
```

`NewSudoku` accepts exactly 81 characters: digits 1–9, with `0` or `.` for
empty cells. It rejects conflicting clues and cells with no candidates;
it does not prove that the puzzle has a unique solution.

For interactive use, call `SolveStep()` repeatedly and render each result.
For benchmarks, `SolveAutomatically()` skips history allocation.

Candidate eliminations must be preserved between steps. Recalculating every
candidate from the board would undo deductions such as Pointing or Hidden Pairs.

## Web interface and API

Run `go run ./cmd/web` from the project root. The website and API listen on
`http://127.0.0.1:8080`. Options:

```sh
go run ./cmd/web -addr 127.0.0.1:8080 -database sudoku-3m.csv
```

Open `http://127.0.0.1:8080` in a browser. The interface supports:

- Filling any cell in any order, with arrow keys to move and Delete/Backspace to clear. During initial manual puzzle entry, entering a digit advances to the next empty cell in reading order, skipping filled cells and wrapping to earlier blanks. Other typed characters (including 0, space, and period) or Enter skip to the next empty cell without changing its value. Arrow keys and Tab retain normal navigation; Delete/Backspace clear a cell without advancing. Random/photo puzzles, editing after a solve, and solution playback do not auto-advance.
- Validating clues and highlighting conflicting cells.
- Loading a random database puzzle.
- Solving and exploring history using Previous/Next, First/Last, or a slider.
- Viewing candidate marks, new placements, and crossed-out eliminated candidates.
- Returning to the original clues for editing, or clearing the board.

The HTML, CSS, and JavaScript in `web/` are embedded in the Go executable.
Restart `go run ./cmd/web` after changing frontend files. No Node build or
separate frontend server is required.

Benchmarks remain available only through the terminal app (`go run .`).
The web server indexes CSV record offsets once at startup and reads a random
record on demand. It does not return the database solution. If the CSV is
missing or unusable, manual validation and solving still work; random returns 503.

### Validate a puzzle

`POST /api/validate` accepts `{"puzzle":"<81 characters>"}`.
Digits 1–9 are clues; `0` and `.` are empty. A future grid can be filled in any
order, then serialized row by row for this request.

```json
{"valid":false,"issues":[{"message":"duplicate 1 in Row 1","cells":[{"row":0,"col":0},{"row":0,"col":1}]}]}
```

The example shows one issue; the response includes all detected conflicts.
Coordinates are zero-based. Conflicting clues, invalid characters, and empty
cells with no candidates are reported. A conflict-free board returns
`{"valid":true,"issues":[]}`. This means **no local conflicts found**, not proof
of solvability or a unique solution. An empty board passes these checks.
Validation results use HTTP 200; malformed JSON uses HTTP 400.

### Solve a puzzle

`POST /api/solve` accepts the same request. Invalid clues return HTTP 400 with
`valid:false` and `issues`. A valid request returns:

```text
{
  "solved": true or false,
  "status": "solved" or "stuck",
  "states": [
    {"board": <9×9 array>, "candidates": <9×9 masks>, "step": null},
    {"board": <9×9 array>, "candidates": <9×9 masks>, "step": <applied step>}
  ]
}
```

Each step includes `technique`, `explanation`, `details`, `row`, `col`, `number`,
`beforeCandidates`, and `eliminations`. A placement has `number` from 1–9;
a candidate-elimination step uses its `eliminations` list instead. Each elimination
has `row`, `col`, and `number`. Candidate masks preserve bit positions 1–9.
The first state is always the original board. `stuck` means supported techniques
made no further progress; it does not mean the puzzle is unsolvable.

### Load a random puzzle

`GET /api/puzzles/random` returns:

```text
{"id":"1","puzzle":"<81 characters>","difficulty":"2.2"}
```

This only loads the puzzle. Send it to `/api/solve` when ready to generate history.

JSON requests have a 4096-byte limit and must contain one object with a `puzzle`
string. Unexpected fields are rejected. General API errors use `{"error":"..."}`.
HTTP routing returns 404 for unknown paths and 405 for unsupported methods.
Implementation: `cmd/web/main.go` starts the server; `internal/webapi` contains
handlers and database access; `solver/validation.go` shares validation with the CLI.

### Visual deduction guides

The Guides checkbox controls highlighted evidence cells, dashed outlines around
focus rows/columns/boxes, and connections to eliminated candidates. Teal marks
identify evidence; coral rings and crossed-out digits identify removals. Hidden
subsets highlight their reserved digits and the other candidates removed from
those same cells. Placement guides now connect the actual already-filled blocking digit to each
ruled-out cell. Amber crossed-out candidates show an existing restriction, not
an elimination being performed by the placement. A separate **Peer Elimination**
step follows when the new placement removes candidates from peers; its coral
marks and arrows originate at that newly placed digit. Placements that remove
no peer candidates do not create an empty cleanup step. With candidates hidden, rings and crosses still mark
target cells. The SVG overlay resizes with the board.

Steps may include `visual` with `sources` (cell coordinates), `digits` (the digits
supporting the deduction), `links` (exact `from`, `to`, `number`, and shared
`unit`), `restrictions` (already excluded candidates), and `units` (`kind`: `row`, `column`, or `box`, plus a
zero-based `index`). These fields are recorded by the techniques themselves;
the UI does not infer evidence from explanation strings.

### Importing a Sudoku photo

Use **Import a photo** to choose an image, or **Take photo** to use a phone's
camera picker. The browser decodes the photo locally; images are not uploaded
to the Go API or an OCR service.

1. Rotate the photo if needed, then drag handles 1–4 clockwise to the outer grid
   corners (top left, top right, bottom right, bottom left). Percentage fields
   offer an accessible alternative to dragging. The guide lines should match
   all nine rows and columns.
2. Choose **Read the grid**. Perspective correction straightens the selected
   grid, grid-line trimming and connected components isolate printed digits,
   and Tesseract.js reads nonempty cells.
3. Review the detected 9×9 board against the photo. Amber means uncertain or
   unreadable; red indicates duplicate clues. Edit or clear any cell, and check
   blanks for missed digits. Choose **Use this puzzle** when ready.
4. The main board remains editable. Validate before solving.

Recognition targets clear, printed, upright 9×9 puzzles. Handwriting, tiny
pencil marks, blur, strong shadows, cropped digits, and incorrect corner
alignment can cause mistakes; no OCR result is automatically submitted to solve.

Tesseract.js 6.0.1 and its worker/WASM/language data download from their public
CDNs on first use, so photo recognition requires internet access to load those
resources. No paid API, secret key, Node build, or Go OCR dependency is needed.
The normal manual-entry and solver features do not depend on OCR downloads.
The file picker accepts browser-supported images up to 20 MB; use JPEG/PNG/WebP
if a phone-specific format cannot be decoded. Closing the dialog cancels an
active recognition job without replacing the current puzzle.

Solving uses a compact, single-row bottom navigation rail with previous/next, timeline, and Edit. On phones the first/last buttons are omitted; the timeline can jump to either end.
On phones, the board fits the available viewport above the dock, and the last
row remains visible when details expand. The elimination summary is inside
**Why this move?**, alongside detailed reasoning; that section scrolls separately
when needed, without moving the navigation controls. Returning to editing restores the
puzzle-entry actions. Detailed reasoning is collapsed by default. Color meanings appear beside the step count. The Godoku title bar disappears once the board contains a puzzle and returns when the board is cleared. Puzzle actions, photo import, and board settings share one horizontal toolbar above the grid; on narrow screens the toolbar scrolls sideways instead of wrapping.

### Placement and elimination sequencing

`SolveStep()` first applies any pending peer cleanup. Otherwise it applies one
technique. A placement fills its cell and clears that cell's candidates; the
other cells retain their candidates until the following `Peer Elimination`
step. No further deduction runs on that intermediate state. `kind` identifies
`placement` versus `elimination` steps. History records both phases and preserves
the pending update when restoring an intermediate snapshot.

`SolveAutomatically()` continues to propagate placements immediately for terminal
benchmarks. Standalone `Apply...` calls and `Place()` also retain immediate
propagation; the separate teaching phases are controlled by `SolveStep()`.
