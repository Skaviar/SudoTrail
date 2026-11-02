# Godoku

A Go Sudoku solver that explains one logical move at a time.

Run the web UI with `make dev`, or benchmark a CSV with
`make benchmark WORKERS=8`. Keep `sudoku-3m.csv` in the project root for
benchmarks; the website uses its embedded practice bank. Run `go run .` to see
command-line help. Use `make check` to run Go tests and `go vet`.

## Quick commands

Run `make` to list the available commands. Go remains the only build dependency;
`make` provides shortcuts for the existing commands.

| Command | What it does |
| --- | --- |
| `make dev` | Rebuild WASM and serve the static website at http://127.0.0.1:8080 |
| `make build` | Generate `dist/` for Cloudflare deployment |
| `make web` | Run the local Go HTTP API and website |
| `make benchmark WORKERS=8` | Benchmark the entire CSV with 8 workers |
| `make rate WORKERS=8` | Export puzzle difficulties to a new `sudoku-rated.csv` |
| `make bank PER_LEVEL=500` | Rebuild the browser practice bank from `sudoku-rated.csv` |
| `make test` | Run Go tests |
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

## Terminal benchmark

Run the full database:

```sh
go run . -benchmark -workers 8
# Equivalent:
make benchmark WORKERS=8
```

For a quick run, process only the first 10,000 puzzle records:

```sh
go run . -benchmark -workers 8 -limit 10000
make benchmark WORKERS=8 LIMIT=10000
```

Flags:

| Flag | Meaning |
| --- | --- |
| `-workers N` | Number of concurrent benchmark/rating goroutines; defaults to `runtime.GOMAXPROCS(0)` |
| `-benchmark` | Run the benchmark and exit |
| `-rate` | Export difficulty ratings and exit |
| `-database PATH` | CSV path; defaults to `sudoku-3m.csv` |
| `-limit N` | Process the first N data records; `0` means the whole file |
| `-output PATH` | New rating CSV path; defaults to `sudoku-rated.csv` and must not already exist |

Choose either `-benchmark` or `-rate` when running a command. With no arguments,
`go run .` prints help and exits. Workers must be positive; limits cannot be negative.

The benchmark counts selected records first, then streams them through a
bounded worker queue. It does not load the 3-million-puzzle dataset into memory.
Each worker has its own Sudoku state and uses `SolveAutomatically()` without
allocating a step history. One goroutine collects results and updates progress
every five seconds, including throughput and ETA.

The final summary reports solved, unsolved, and invalid records separately.
**Solve rate = solved / processed × 100**; invalid records are included in the
denominator. A complete run with no limit therefore gives the percentage
solved from the entire CSV. Headers and blank CSV lines are excluded. Malformed
CSV syntax or read errors stop the run rather than silently skipping data.
Press **Ctrl+C** to stop early and print a partial summary; its solve rate
applies only to the processed records. Elapsed time and throughput measure the
solving pass, excluding the initial counting pass.

All solves use logical techniques. Backtracking is used only to check whether
the original puzzle has exactly one solution, as required by uniqueness
techniques. It never supplies placements or candidate eliminations. When the
available techniques cannot advance a puzzle, it remains unsolved.

## Difficulty export

Rate the full database and create a new file:

```sh
make rate WORKERS=8
# Equivalent:
go run . -rate -workers 8 -output sudoku-rated.csv
```

For a small trial, choose a different output filename and a limit:

```sh
make rate WORKERS=8 LIMIT=10000 OUTPUT=sudoku-rated-test.csv
```

The source CSV is read only. Existing output files are never overwritten;
choose an unused output path for another run.

The current policy is **`godoku-v1`**:

| Category | Techniques allowed before moving to the next tier |
| --- | --- |
| Easy | Full House, Naked Single, Hidden Single |
| Medium | Pointing, Claiming, Naked/Hidden Pairs and Triples |
| Hard | Quads, X-Wing/Swordfish/Jellyfish, Skyscraper/2-String Kite/Turbot Fish/Empty Rectangle, XY/XYZ/W-Wings, Remote Pairs, Simple Coloring, UR types 1/2, BUG+1 |
| Expert | Finned/Sashimi and larger basic fish, complex/Siamese fish, advanced or missing-candidate rectangles, Multi-Coloring/Medusa, chains/ALS, forcing chains/nets |

After **every deduction**, the rating solver restarts at Easy. It exhausts
all lower tiers before trying an Expert move. The web practice solver follows
this same order so its hints agree with the difficulty labels and technique
filters. The solver library's default teaching order can produce a different
path. These are Godoku-specific ratings of the path found, not a proof that
no easier solution path exists.

The label is the hardest tier used. A separate **score** adds the technique
weight for every deduction; it distinguishes the amount of work within a
category without promoting a long singles-only puzzle to Expert. For example,
Full House/Naked Single cost 1, Hidden Single 2, Naked Pair 10, X-Wing 40,
XY-Wing 60, 3D Medusa 180, Forcing Chain 500, and Forcing Net 800. The complete
weights and fish-name rules are in [solver/rating.go](solver/rating.go).
Changing the policy requires updating its version.

Each placement or elimination pattern counts as one logical step. Automatic
peer-candidate cleanup counts as zero steps and zero points. Source difficulty
and supplied solution strings do not influence the rating. Backtracking remains
restricted to uniqueness checks.

The export has these columns:

```csv
id,puzzle,solution,clues,difficulty,original_difficulty,score,hardest_technique,logical_steps,technique_counts,rating_status,rating_version,rating_error
```

`technique_counts` is a JSON object inside a properly quoted CSV field.
`rating_status` is `solved`, `unsolved`, or `invalid`. Only solved puzzles get
an Easy/Medium/Hard/Expert label. Unsolved rows retain any partial score and
technique counts; invalid rows include a validation error. A completed grid
needs zero deductions and is Easy with score 0.

Output rows retain input order, regardless of worker count. Both jobs and
out-of-order results are bounded in memory, and no solving histories are
retained. Progress includes each difficulty count and ETA; the final summary
prints counts and percentages of processed records. **Ctrl+C** flushes an
ordered partial CSV containing the written rows. CSV/I/O failures are reported
instead of silently dropping records.

The first five columns remain compatible with benchmarks and the HTTP catalog:

```sh
go run . -benchmark -database sudoku-rated.csv
go run ./cmd/web -database sudoku-rated.csv
```

The browser's balanced practice bank is a separate checked-in catalog. Exporting
this file does not automatically replace it; see **Browser practice catalog**
below for the generation command.

Split a rated catalog into separate difficulty files with Python 3.9 or newer
(no packages to install):

```sh
python3 scripts/split_difficulties.py sudoku-rated.csv
# Or choose another destination:
python3 scripts/split_difficulties.py sudoku-rated.csv --output-dir my-puzzles
```

The default directory is `puzzles-by-difficulty/`, containing `easy.csv`,
`medium.csv`, `hard.csv`, and `expert.csv`. Every file retains the original
header, all columns, and the relative order of its puzzles. Unsolved, invalid,
or unlabeled rows go into `unrated.csv` only when such rows exist. The script
streams the input, reports counts, and refuses to overwrite existing files.
Unknown difficulty labels or malformed records stop with an error; partial
files may remain, so use a new output directory for another run.

Test the splitter with `python3 -m unittest discover -s scripts -p 'test_*.py'`.

### Browser practice catalog

The website includes **500 Easy, 500 Medium, 500 Hard, and 500 Expert puzzles**.
The start screen offers **Your puzzle** first, followed by **Practice techniques**,
under **Crack the grid.**, with a short line identifying the step-by-step Sudoku
solutions. The start screen uses a dark puzzle-entry card and a light practice
card; the decorative grid stays hidden on phones to keep the controls in view.
Typography uses system fonts, lighter interface text, and semibold tabular grid
digits, with muted sage accents to keep the puzzle visually prominent.
Short captions summarize the levels; a technique description appears only when
you select a specific focus. Only techniques actually used in matching puzzles
are offered, with their available puzzle counts. A level describes the hardest technique used;
it can also contain techniques from earlier levels.

Choosing a level starts an editable puzzle. Choosing a technique opens a
challenge immediately before the solver's first use of it, with candidate marks
visible and the deduction hidden. **Reveal hint** shows the move and its visual
evidence; **View all steps** returns to the original puzzle's history. **Edit**
returns to the original editable clues.

To rebuild the bank from your rated CSV:

```sh
make bank
# Use more puzzles per level or a different rated input:
make bank PER_LEVEL=1000 OUTPUT=sudoku-rated.csv
# Control the output path and sampling seed directly:
go run ./cmd/buildbank -in sudoku-rated.csv -out internal/puzzlebank/puzzles.json -per-level 500 -seed 20261101
```

Generation samples across the full rated input with a fixed seed, verifies each
selected puzzle against its reference solution, and records the techniques used
by web playback. It requires the current `godoku-v1` rating model and enough
solved puzzles in every level. It replaces the bank after verification completes;
the rated CSV remains unchanged. No solution strings are written into the bank.
Rebuild the website with `make build` after changing it.

The current 2,000-puzzle bank is about **498 KiB of JSON**. The compiled WASM is
about **4.30 MiB** (sizes vary with Go versions), so there is room to expand the
catalog. These are file sizes, not browser memory measurements. Normal website
builds use the checked-in bank and do not read the large CSV.

## Logical techniques

The terminal, HTTP API, and WASM build use the same techniques. Simpler moves
run first; the advanced stage runs only when those moves stall.

| Family | Implemented |
| --- | --- |
| Singles and subsets | Full House, Naked/Hidden Singles, Naked/Hidden Pairs, Triples and Quads; Pointing and Claiming |
| Basic fish | X-Wing through Leviathan (sizes 2–7), including finned and sashimi variants |
| Complex fish | Franken and Mutant fish (sizes 2–7), fins, endo fins, cannibalistic eliminations, and Siamese combinations |
| Single-digit patterns | Skyscraper, 2-String Kite, Turbot Fish, Empty Rectangle |
| Wings | XY-Wing, XYZ-Wing, W-Wing |
| Uniqueness | Unique Rectangle types 1–6, Hidden Rectangle, Avoidable Rectangle types 1/2, missing-candidate rectangles, BUG+1 |
| Coloring | Remote Pairs, Simple Coloring, Multi-Coloring, 3D Medusa (cell and unit conflicts/traps) |
| Chains and loops | X-Chains, XY-Chains, Alternating Inference Chains, including endpoint and Nice Loop eliminations |
| Almost locked sets | Singly/doubly linked ALS-XZ, ALS-XY-Wing, ALS Chains, Death Blossom |
| Forcing | Contradictions and common conclusions from Forcing Chains and Forcing Nets |

**Forcing is an explained logical proof.** Each branch starts with one premise,
then follows links or propagates singles. Nets can combine several earlier
deductions. Branches never create further assumptions, recursively explore
completions, or use a completed solution. The step contains the dependencies
needed to check the conclusion. In the website, “Why this move?” lets you view
one proof branch at a time. Placements and their peer eliminations remain
separate steps.

Complex fish detection is deliberately bounded to keep browser solves and
large benchmarks responsive: each digit/size has a budget of 3,000 cover-tree
nodes, with smaller candidate houses considered first. This can miss a valid
fish; it cannot make a returned fish deduction unsound. Multi-Coloring combines
two components at a time; AICs handle longer connections. The implemented
catalog is not a claim that every possible Sudoku or every variant of a named
pattern will be found.

All uniqueness rules first verify the original clues. Missing rectangle
candidates must be allowed by those clues, and Avoidable Rectangles cannot
include an original given. The uniqueness checker returns a verdict only.

The rules follow the explanations in [HoDoKu's technique guide](https://hodoku.sourceforge.net/en/techniques.php)
and Andrew Stuart's [3D Medusa guide](https://www.sudokuwiki.org/3d_medusa).

For the advanced implementation, read `solver/inference.go` (candidate links),
`solver/chains.go` (alternating paths), then `solver/forcing.go` (proof branches).
The other families live in `rectangles.go`, `complex_fish.go`, `medusa.go`, and
`als.go`. `topology.go` stores sets of cells; these are distinct from the digit
bitmasks in `candidates.go`.

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

The command entry points are:

| Command | Mode |
| --- | --- |
| `go run . -benchmark` | Full-database benchmark |
| `go run . -rate` | Difficulty export |
| `go run ./cmd/web` | Embedded UI with the local Go HTTP API/full CSV |
| `go run ./cmd/buildweb` | Generate a deployable static WASM website |
| `go run ./cmd/web -static dist` | Preview static WASM output |

The HTTP API and WASM entry point share `internal/puzzleapp`, so snapshots,
visual guides, errors, and solved/stuck behavior use the same solver logic.
`internal/puzzlebank` owns the balanced practice catalog; `cmd/wasm` bridges Go to
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
- `main.go`: command-line flags for benchmarking and difficulty export.
- `benchmark.go` and `rating.go`: streaming CSV processing, concurrent workers,
  progress reporting, and difficulty export. The solver package has no terminal
  dependencies.

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

Open `http://127.0.0.1:8080` in a browser. Start by choosing a difficulty and an
optional technique, **Enter numbers**, or **Import a photo**. The interface supports:

- Filling any cell in any order, with arrow keys to move and Delete/Backspace to clear. During initial manual or photo puzzle entry, entering a digit advances to the next empty cell in reading order, skipping filled cells and wrapping to earlier blanks. Other typed characters (including 0, space, and period) or Enter skip to the next empty cell without changing its value. Arrow keys and Tab retain normal navigation; Delete/Backspace clear a cell without advancing. Random puzzles, editing after a solve, and solution playback do not auto-advance.
- Validating clues and highlighting conflicting cells.
- Loading a practice puzzle by difficulty, optionally focused on one technique.
- Solving and exploring history using Previous/Next, First/Last, or a slider.
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
`node --test web/session_test.mjs web/training_test.mjs`
when Node is available; Node is not required to build or run the app.

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

Use **Import a photo** to choose an image, or **Take photo** to use a phone's
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
4. The photo remains visible alongside the editable main board (above it on
   phones). Both review and initial photo entry advance to the next blank when
   you type a digit. Validate before solving; Solve ends automatic advancing.

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

### Naked subsets

The solver supports Naked Pairs, Triples, and Quadruples in rows, columns, and
boxes. It tries combinations of 2–4 empty cells and requires exactly that many
distinct candidates across the selected cells. Candidates can differ between
cells, for example `{2,5}`, `{5,8}`, `{2,8}` for a triple.

Reserved digits are removed only from other cells in shared units. A locked
pair/triple sharing both a line and a box applies to both, deduplicating targets.
No placement is made, and subsets with no actual eliminations are skipped.
The UI highlights the selected cells/digits and shared units, with lines to
removed candidates originating only at source cells that allow that digit.

Smaller subsets run first: Naked Pair, Hidden Pair, Naked Triple, Hidden Triple,
Naked Quadruple, Hidden Quadruple (after singles and intersections).
Implementation: `solver/naked_subsets.go`.

### Fish, wings, and single-digit patterns

The solver also supports these ten techniques:

- **X-Wing, Swordfish, Jellyfish** (`solver/fish.go`): for one digit, reserve
  2, 3, or 4 columns for the same number of rows, or vice versa. Intersections
  may be sparse. Candidates outside the selected rows/columns are eliminated.
- **XY-Wing, XYZ-Wing, W-Wing** (`solver/wings.go`): use connected candidate
  cells to prove at least one witness must contain a digit. XYZ-Wing targets
  must see the pivot as well as both wings. W-Wing requires two identical
  bivalue cells connected through a unit with only two places for one digit.
- **Skyscraper, 2-String Kite, Turbot Fish, Empty Rectangle**
  (`solver/single_digit.go`): combine restricted locations of a single digit.
  Named Skyscrapers and Kites run before the more general Turbot detector.
  Empty Rectangles use at least three candidates in the box; the equivalent
  two-candidate patterns are handled by Turbot Fish.

All are candidate-elimination steps, with explanations and explicit visual
evidence. Solid teal lines join the only two possible locations in a unit;
dashed teal lines connect candidates that cannot both be true. Coral lines
connect the supporting candidates to removals. These candidate links are
separate from arrows originating at already placed numbers.

After singles, intersections, and subsets, the order is X-Wing, Skyscraper,
2-String Kite, Turbot Fish, Empty Rectangle, XY-Wing, XYZ-Wing, W-Wing,
Swordfish, Jellyfish. A successful elimination returns immediately so the next
step checks simpler techniques again. These methods use the same Go code in
the terminal, HTTP API, and browser WASM build.

Tests cover both orientations, sparse fish, broken patterns, visual evidence,
known puzzle solutions, and independently enumerated single-digit placements.
Definitions: [basic fish](https://hodoku.sourceforge.net/en/tech_fishb.php),
[wings](https://hodoku.sourceforge.net/en/tech_wings.php), and
[single-digit patterns](https://hodoku.sourceforge.net/en/tech_sdp.php).

### Additional fish, coloring, and uniqueness

The next logical group runs after the existing techniques, in this order:

1. **Finned/Sashimi X-Wing** (`solver/finned_fish.go`): a fish with uncovered
   candidates (fins). Only cover candidates outside the base units that see
   every fin can be removed. Fins are amber, and their shared box is outlined.
2. **Remote Pairs** (`solver/coloring.go`): connected identical two-candidate
   cells must alternate values. Cells seeing opposite assignments lose the pair
   digits. Both candidates have explicit opposite color markings.
3. **Simple Coloring** (`solver/coloring.go`): conjugate pairs create alternate
   color assignments for one digit. Color Trap removes an outside candidate
   seeing both colors. Color Wrap removes a whole color when two candidates
   of that color share a unit. Contradictory odd-cycle components are skipped.
4. **Unique Rectangle Types 1–2** (`solver/uniqueness.go`): complete rectangles
   in exactly two rows, two columns, and two boxes. Type 1 removes the rectangle
   digits from the sole cell with extra candidates; Type 2 removes the shared
   extra digit from cells seeing both adjacent extra-candidate cells. Missing
   candidates and other rectangle types are not part of these detectors.
5. **BUG+1** (`solver/uniqueness.go`): every empty cell except one is bivalue;
   the remaining cell is trivalue. Removing its extra candidate must produce
   exactly two locations per unplaced digit in every unit. That extra digit
   is placed, followed by a separate peer-elimination step.
6. **Larger basic fish** (`solver/fish.go`): Squirmbag (size 5), Whale (6), and
   Leviathan (7), in either orientation, using the existing counting rule.
7. **Larger Finned/Sashimi fish** (`solver/finned_fish.go`): Swordfish (3),
   Jellyfish (4), Squirmbag (5), Whale (6), and Leviathan (7), tried in that
   order after the existing methods. Each uses the same fin rule as X-Wing.

`ApplyFinnedFish(s, size)` supports sizes 2–7 in rows or columns;
`ApplyFinnedXWing(s)` remains the size-2 entry point. A fin-free core is called
**Sashimi** when it reduces to a single or a smaller fish, including a cover
with only one possible location. Sparse intersections alone do not make it
Sashimi. All fins must share a box, and eliminations stay inside that box,
outside the selected base units and inside the selected cover units.

To limit benchmark overhead, candidate positions and unit combinations are
shared by the fish detectors. For each possible fin box, every candidate
outside that box must be covered, leaving at most eight cover combinations.
Tests independently enumerate legal placements of the fish digit, including
sparse patterns, multiple fins, both orientations, and randomized candidates.

Uniqueness deductions require proof that the **original submitted clues** have
exactly one solution. `HasUniqueSolution()` (`solver/uniqueness_check.go`) uses
backtracking capped at two completions and caches only the uniqueness verdict.
The check ignores later placements and candidate eliminations and never returns
a completed board. Multiple-solution and unsatisfiable submissions skip these
methods. Snapshots preserve original clues when restored.

Backtracking is used only for this uniqueness check. All solving steps come
from logical techniques; there is no search fallback.

Definitions: [finned fish](https://hodoku.sourceforge.net/en/tech_fishfs.php),
[coloring](https://hodoku.sourceforge.net/en/tech_col.php),
[remote pairs](https://hodoku.sourceforge.net/en/tech_chains.php), and
[uniqueness](https://hodoku.sourceforge.net/en/tech_ur.php).

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
