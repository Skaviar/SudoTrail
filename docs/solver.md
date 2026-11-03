# Solver reference

[Back to the README](../README.md).

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

## Technique notes

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
