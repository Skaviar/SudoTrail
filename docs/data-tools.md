# Data tools

Run commands from the project root. [Back to the README](../README.md).

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
path. These are Sudotrail-specific ratings of the path found, not a proof that
no easier solution path exists.

The label is the hardest tier used. A separate **score** adds the technique
weight for every deduction; it distinguishes the amount of work within a
category without promoting a long singles-only puzzle to Expert. For example,
Full House/Naked Single cost 1, Hidden Single 2, Naked Pair 10, X-Wing 40,
XY-Wing 60, 3D Medusa 180, Forcing Chain 500, and Forcing Net 800. The complete
weights and fish-name rules are in [solver/rating.go](../solver/rating.go).
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
Choose a level under **Practice techniques**, optionally with a technique focus.
Only techniques used in matching puzzles are offered, with their available
puzzle counts. A level describes the hardest technique used; its puzzles can
also use techniques from earlier levels.

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
