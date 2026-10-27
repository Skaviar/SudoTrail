package solver

import (
	"encoding/json"
	"os"
	"testing"
)

// These reference puzzles/solutions come from the existing sudoku-3m catalog.
// They exercise complete solving histories, including interactions with the
// earlier techniques, rather than only testing isolated constructed patterns.
func TestAdvancedPatternsPreserveReferenceSolutions(t *testing.T) {
	data, err := os.ReadFile("testdata/advanced_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct{ Technique, Puzzle, Solution string }
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Technique, func(t *testing.T) {
			s, err := NewSudoku(Puzzle{Puzzle: fixture.Puzzle})
			if err != nil {
				t.Fatal(err)
			}
			history := s.Solve()
			found := false
			for i, entry := range history.Entries {
				found = found || entry.Step.Technique == fixture.Technique
				for row := range 9 {
					for col := range 9 {
						number := int8(fixture.Solution[row*9+col] - '0')
						value := entry.State.Board[row][col]
						if value != 0 && value != number || value == 0 && !hasCandidate(entry.State.Candidates[row][col], int(number)) {
							t.Fatalf("step %d (%s) discarded solution digit %d at %s", i, entry.Step.Technique, number, cellName(row, col))
						}
					}
				}
			}
			if !found {
				t.Fatal("reference no longer exercises its named technique")
			}
			automatic, _ := NewSudoku(Puzzle{Puzzle: fixture.Puzzle})
			automatic.SolveAutomatically()
			if CaptureBoardState(automatic) != CaptureBoardState(s) {
				t.Fatal("automatic and history-based solving disagree")
			}
		})
	}
}

// Independently enumerate all placements of one digit that satisfy rows,
// columns, and boxes. No eliminated candidate may occur in ANY placement.
// This also covers Jellyfish, which is rare in the random reference sample.
func TestSingleDigitEliminationsAgainstAllPlacements(t *testing.T) {
	check := func(t *testing.T, s *Sudoku, apply func(*Sudoku) (Step, bool)) {
		t.Helper()
		before := CaptureBoardState(s)
		step, ok := apply(s)
		if !ok {
			t.Fatal("fixture has no elimination")
		}
		var placement [9]int
		count := 0
		var visit func(int, uint16, uint16)
		visit = func(row int, columns, boxes uint16) {
			if row == 9 {
				count++
				for _, e := range step.Eliminations {
					if placement[e.Row] == e.Col {
						t.Fatalf("%s removed a candidate used in a legal digit placement: %+v", step.Technique, e)
					}
				}
				return
			}
			for col := range 9 {
				box := row/3*3 + col/3
				if columns&(1<<col) != 0 || boxes&(1<<box) != 0 || !hasCandidate(before.Candidates[row][col], 7) {
					continue
				}
				placement[row] = col
				visit(row+1, columns|(1<<col), boxes|(1<<box))
			}
		}
		visit(0, 0, 0)
		if count == 0 {
			t.Fatal("fixture has no legal digit placement")
		}
	}
	for size := 2; size <= 4; size++ {
		for _, transposed := range []bool{false, true} {
			check(t, fishFixture(size, transposed), func(s *Sudoku) (Step, bool) { return ApplyBasicFish(s, size) })
		}
	}
	for _, tt := range []struct {
		name  string
		apply func(*Sudoku) (Step, bool)
	}{
		{"Skyscraper", ApplySkyscraper}, {"2-String Kite", ApplyTwoStringKite},
		{"Turbot Fish", ApplyTurbotFish}, {"Empty Rectangle", ApplyEmptyRectangle},
	} {
		for _, transposed := range []bool{false, true} {
			t.Run(tt.name, func(t *testing.T) { check(t, singleDigitFixture(tt.name, transposed), tt.apply) })
		}
	}
}
