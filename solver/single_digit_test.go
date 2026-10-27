package solver

import (
	"fmt"
	"testing"
)

func restrictDigit(s *Sudoku, u unit, allowed []Cell, number int) {
	for _, cell := range u.cells {
		if !containsCell(allowed, cell) {
			s.Candidates[cell.Row][cell.Col] = removeCandidate(s.Candidates[cell.Row][cell.Col], number)
		}
	}
}

func singleDigitFixture(name string, transposed bool) *Sudoku {
	s := patternSudoku()
	units := allUnits()
	switch name {
	case "Skyscraper":
		restrictDigit(s, units[1], []Cell{{1, 0}, {1, 4}}, 7)
		restrictDigit(s, units[7], []Cell{{7, 0}, {7, 3}}, 7)
	case "2-String Kite":
		restrictDigit(s, units[7], []Cell{{7, 3}, {7, 8}}, 7)
		restrictDigit(s, units[9+6], []Cell{{1, 6}, {8, 6}}, 7)
	case "Turbot Fish":
		restrictDigit(s, units[0], []Cell{{0, 0}, {0, 4}}, 7)
		restrictDigit(s, units[18+4], []Cell{{3, 4}, {4, 5}}, 7)
	case "Empty Rectangle":
		restrictDigit(s, units[18+4], []Cell{{3, 4}, {3, 5}, {5, 5}}, 7)
		restrictDigit(s, units[9+1], []Cell{{3, 1}, {7, 1}}, 7)
	}
	if transposed {
		for row := range 9 {
			for col := row + 1; col < 9; col++ {
				s.Candidates[row][col], s.Candidates[col][row] = s.Candidates[col][row], s.Candidates[row][col]
			}
		}
	}
	return s
}

func TestSingleDigitPatterns(t *testing.T) {
	for _, tt := range []struct {
		name  string
		apply func(*Sudoku) (Step, bool)
	}{
		{"Skyscraper", ApplySkyscraper}, {"2-String Kite", ApplyTwoStringKite},
		{"Turbot Fish", ApplyTurbotFish}, {"Empty Rectangle", ApplyEmptyRectangle},
	} {
		for _, transposed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/transposed=%v", tt.name, transposed), func(t *testing.T) {
				s := singleDigitFixture(tt.name, transposed)
				before := CaptureBoardState(s)
				step, ok := tt.apply(s)
				if !ok {
					t.Fatal("pattern not found")
				}
				assertPatternStep(t, before, s, step, tt.name)
				for _, e := range step.Eliminations {
					if e.Number != 7 {
						t.Fatal("pattern removed an unrelated digit")
					}
				}
				before.Restore(s)
				step, ok = s.SolveStep()
				if !ok {
					t.Fatal("pipeline did not find pattern")
				}
				assertPatternStep(t, before, s, step, tt.name)
			})
		}
	}
}

func TestSingleDigitPatternsRejectBrokenLinks(t *testing.T) {
	for _, tt := range []struct {
		name  string
		apply func(*Sudoku) (Step, bool)
		extra Cell
	}{
		{"Skyscraper", ApplySkyscraper, Cell{1, 8}},
		{"2-String Kite", ApplyTwoStringKite, Cell{7, 0}},
		{"Turbot Fish", ApplyTurbotFish, Cell{0, 8}},
		{"Empty Rectangle", ApplyEmptyRectangle, Cell{0, 1}},
		{"Empty Rectangle", ApplyEmptyRectangle, Cell{4, 3}},
	} {
		t.Run(fmt.Sprint(tt.name, tt.extra), func(t *testing.T) {
			s := singleDigitFixture(tt.name, false)
			s.Candidates[tt.extra.Row][tt.extra.Col] |= 1 << 7
			before := CaptureBoardState(s)
			if _, ok := tt.apply(s); ok || CaptureBoardState(s) != before {
				t.Fatal("pattern accepted an extra candidate that breaks its premise")
			}
		})
	}
}

func TestPatternsWithNoEliminationsDoNothing(t *testing.T) {
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyXYWing, ApplyXYZWing, ApplyWWing, ApplySkyscraper, ApplyTwoStringKite, ApplyTurbotFish, ApplyEmptyRectangle} {
		s := patternSudoku()
		before := CaptureBoardState(s)
		if _, ok := apply(s); ok || CaptureBoardState(s) != before {
			t.Fatal("pattern reported a deduction on an unconstrained grid")
		}
	}
}
