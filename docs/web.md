# Web interface and API

[Back to the README](../README.md).

Run `go run ./cmd/web` from the project root. The website and API listen on
`http://127.0.0.1:8080`. Options:

```sh
go run ./cmd/web -addr 127.0.0.1:8080 -database sudoku-3m.csv
```

Open `http://127.0.0.1:8080` in a browser. Start by choosing a difficulty and an
optional technique, **Enter numbers**, or **Import a photo**. The interface supports:

- Filling empty cells in any order, with arrow keys to move and Delete/Backspace to clear. During initial manual or photo puzzle entry, entering a digit advances to the next empty cell in reading order, skipping filled cells and wrapping to earlier blanks. Other typed characters (including 0, space, and period) or Enter skip to the next empty cell without changing its value. Arrow keys and Tab retain normal navigation; Delete/Backspace clear a cell without advancing. Random puzzles, editing after a solve, and solution playback do not auto-advance.
- Automatically checking the grid as numbers are entered or cleared, highlighting
  conflicting clues and empty cells with no possible digits. Checks run in the
  background; correcting a conflict clears its warning.
- Loading a practice puzzle by difficulty, optionally focused on one technique.
- Choosing **Solve step by step** and exploring history using Previous/Next,
  First/Last, or a slider.
- Viewing candidate marks, new placements, and crossed-out eliminated candidates.
- Opening **Learn technique** beside a hint or selected practice focus for its
  HoDoKu explanation (SudokuWiki for 3D Medusa). Guides open in a new tab.
- Returning to the original clues for editing, or choosing **New puzzle** to
  return to setup.
- Automatically saving the latest puzzle in this browser, including edits,
  clue colors, photo reference, display settings, and the selected solving step.
  Technique challenges also preserve their target and hidden-hint state.
  Refreshing or returning to the site restores it; **New puzzle** deletes the save.

Progress uses `localStorage` on the same browser and site address, with no
account, cookies, or server storage. It does not sync between devices. Solving
history is rebuilt from the saved submission when reopening playback. Clearing
site data removes the save. If the browser blocks saving or cannot fit a photo,
the interface reports it while keeping editing and solving available.
Storage and practice helpers can also be checked with
`node --test web/*_test.mjs`
when Node is available; Node is not required to build or run the app.

On mobile and touch devices, select a cell and use the on-screen 1–9 keypad.
**Clear cell** removes an entry; **Undo** restores edits made during the current
visit (also Ctrl/Cmd+Z). Loaded clues are locked, including after returning from
solution playback. Manually entered clues lock after a successful solve request.

The HTML, CSS, and JavaScript in `web/` are embedded in the Go executable.
Restart `go run ./cmd/web` after changing frontend files. No Node build or
separate frontend server is required.

Benchmarks run through `make benchmark` or `go run . -benchmark`.
The practice library, manual entry, validation, and solving work without a CSV.
The native web server also retains unfiltered full-database random selection:
it indexes CSV offsets at startup and reads one record on demand, without
returning the solution. Only that legacy unfiltered route returns 503 when the
CSV is missing or unusable.

### Validate a puzzle

`POST /api/validate` accepts `{"puzzle":"<81 characters>"}`.
Digits 1–9 are clues; `0` and `.` are empty. Serialize the grid row by row for
this request.

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

### Choose a practice puzzle

`GET /api/puzzles/catalog` returns the available difficulty levels, their puzzle
counts, and technique entries with `name`, `count`, and `difficulty`.

`GET /api/puzzles/random?source=practice&difficulty=hard&technique=XY-Wing`
returns a matching embedded puzzle:

```text
{"id":"123","puzzle":"<81 characters>","difficulty":"hard","techniques":["Hidden Single","XY-Wing", "..."]}
```

The `technique` filter is optional and case-sensitive; URL-encode technique
names containing spaces. `exclude=<id>` avoids immediately repeating that puzzle
when another match exists. Invalid filters or no matches return HTTP 400. No
completed solution is included. These routes work in both native and WASM mode.

### Load from the full database

In native API mode, `GET /api/puzzles/random` without filters uses the CSV and returns:

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
`unit` for placed blockers), `candidateLinks` (candidate connections with
`from`, `to`, `number`, `unit`, and `kind`: `strong`, `weak`, or `elimination`),
`restrictions` (already excluded candidates), and `units` (`kind`: `row`, `column`, or `box`, plus a
zero-based `index`). These fields are recorded by the techniques themselves;
the UI does not infer evidence from explanation strings.
Optional `highlights` identify a candidate by `row`, `col`, `number`, and `role`:
`fin` (amber), `color-a` (blue), or `color-b` (purple). These are copied into
history and cleared when guides are disabled or another step is selected.

### Importing a Sudoku photo

Use **Import a photo** (beta) to choose an image, or **Take photo** to use a phone's
camera picker. The browser decodes the photo locally; images are not uploaded
to the Go API or an OCR service.

1. Rotate the photo if needed, then drag handles 1–4 clockwise to the outer grid
   corners (top left, top right, bottom right, bottom left). The guide lines
   should match all nine rows and columns.
2. Choose **Read the grid**. Perspective correction straightens the selected
   grid, grid-line trimming and connected components isolate printed digits,
   and Tesseract.js reads nonempty cells.
3. Review the detected 9×9 board beside the straightened photo, or below it on
   phones. Crop controls are hidden during review; **Adjust crop** returns to
   them. Amber means uncertain or
   unreadable; red indicates duplicate clues. Edit or clear any cell, and check
   blanks for missed digits. Choose **Use this puzzle** when ready.
4. The photo remains visible alongside the main board (above it on phones).
   Reviewed clues are locked when you choose **Use this puzzle**; empty cells
   remain editable. Correct recognized clues in the review grid before loading.

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

Playback uses a bottom navigation rail with previous/next, a timeline, and
**Back to puzzle**. On phones, the board stays above the rail and detailed reasoning
scrolls separately. **Why this move?** contains the elimination summary and
proof details; color meanings appear beside the step count.
