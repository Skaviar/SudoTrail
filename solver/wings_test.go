package solver

import (
	"fmt"
	"testing"
)

func wingFixture(name string, transposed bool) *Sudoku {
	s := patternSudoku()
	set := func(cell Cell, digits ...int) {
		if transposed {
			cell = Cell{cell.Col, cell.Row}
		}
		s.Candidates[cell.Row][cell.Col] = maskOf(digits...)
	}
	set(Cell{0, 0}, 1, 2)
	set(Cell{0, 4}, 1, 3)
	set(Cell{4, 0}, 2, 3)
	if name == "XYZ-Wing" {
		set(Cell{0, 0}, 1, 2, 3)
		set(Cell{4, 0}, 1, 2, 3, 4, 5, 6, 7, 8, 9)
		set(Cell{1, 1}, 2, 3)
	}
	if name == "W-Wing" {
		s = patternSudoku()
		set(Cell{0, 0}, 1, 3)
		set(Cell{7, 4}, 1, 3)
		for col := range 9 {
			if col != 0 && col != 4 {
				cell := Cell{4, col}
				if transposed {
					cell = Cell{cell.Col, cell.Row}
				}
				s.Candidates[cell.Row][cell.Col] = removeCandidate(AllCandidates, 1)
			}
		}
	}
	return s
}

func TestWings(t *testing.T) {
	for _, tt := range []struct {
		name  string
		apply func(*Sudoku) (Step, bool)
	}{
		{"XY-Wing", ApplyXYWing}, {"XYZ-Wing", ApplyXYZWing}, {"W-Wing", ApplyWWing},
	} {
		for _, transposed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/transposed=%v", tt.name, transposed), func(t *testing.T) {
				s := wingFixture(tt.name, transposed)
				before := CaptureBoardState(s)
				step, ok := tt.apply(s)
				if !ok {
					t.Fatal("wing not found")
				}
				assertPatternStep(t, before, s, step, tt.name)
				for _, e := range step.Eliminations {
					if e.Number != 3 {
						t.Fatal("removed a digit other than the shared wing digit")
					}
				}
				before.Restore(s)
				step, ok = s.SolveStep()
				if !ok {
					t.Fatal("pipeline missed wing")
				}
				assertPatternStep(t, before, s, step, tt.name)
			})
		}
	}
}

func TestXYZWingRequiresTargetToSeePivot(t *testing.T) {
	s := wingFixture("XYZ-Wing", false)
	before := CaptureBoardState(s)
	step, ok := ApplyXYZWing(s)
	if !ok {
		t.Fatal("wing not found")
	}
	assertPatternStep(t, before, s, step, "XYZ-Wing")
	if hasCandidate(s.Candidates[0][1], 3) || !hasCandidate(s.Candidates[1][4], 3) {
		t.Fatal("XYZ-Wing must remove 3 only from cells seeing the pivot AND both wings")
	}
}

func TestWingsRejectIncompletePatterns(t *testing.T) {
	for _, tt := range []struct {
		name         string
		apply        func(*Sudoku) (Step, bool)
		breakPattern func(*Sudoku)
	}{
		{"XY-Wing", ApplyXYWing, func(s *Sudoku) { s.Candidates[4][0] |= 1 << 5 }},
		{"XY-Wing", ApplyXYWing, func(s *Sudoku) { s.Candidates[4][8], s.Candidates[4][0] = s.Candidates[4][0], AllCandidates }},
		{"XYZ-Wing", ApplyXYZWing, func(s *Sudoku) { s.Candidates[1][1] = maskOf(2, 4) }},
		{"W-Wing", ApplyWWing, func(s *Sudoku) { s.Candidates[4][8] |= 1 << 1 }},
		{"W-Wing", ApplyWWing, func(s *Sudoku) { s.Candidates[7][4] = maskOf(1, 4) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := wingFixture(tt.name, false)
			tt.breakPattern(s)
			before := CaptureBoardState(s)
			if _, ok := tt.apply(s); ok || CaptureBoardState(s) != before {
				t.Fatal("accepted an incomplete wing")
			}
		})
	}
}
