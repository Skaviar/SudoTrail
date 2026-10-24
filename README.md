# Godoku

A Go Sudoku solver that explains one logical move at a time.

Run the terminal app with `go run .`. Keep `sudoku-3m.csv` in the project
root for database puzzles. Use `go test ./...` to run the tests.

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
