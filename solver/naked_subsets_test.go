package solver

import (
	"fmt"
	"strings"
	"testing"
)

func maskOf(digits ...int) uint16 {
	var mask uint16
	for _, digit := range digits {
		mask |= 1 << digit
	}
	return mask
}

func TestNakedSubsetsInEveryUnit(t *testing.T) {
	patterns := map[int][]uint16{
		2: {maskOf(2, 7), maskOf(2, 7)},
		3: {maskOf(2, 5), maskOf(5, 8), maskOf(2, 8)},
		4: {maskOf(1, 4), maskOf(4, 6), maskOf(6, 9), maskOf(1, 9)},
	}
	shapes := map[string][]Cell{
		"row":    {{0, 0}, {0, 4}, {0, 8}, {0, 2}},
		"column": {{0, 0}, {4, 0}, {8, 0}, {2, 0}},
		"box":    {{0, 0}, {1, 1}, {2, 2}, {0, 1}},
	}
	for size := 2; size <= 4; size++ {
		for kind, shape := range shapes {
			t.Run(fmt.Sprintf("%s/%d", kind, size), func(t *testing.T) {
				s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
				sources := shape[:size]
				var reserved uint16
				for i, cell := range sources {
					s.Candidates[cell.Row][cell.Col] = patterns[size][i]
					reserved |= patterns[size][i]
				}
				before := CaptureBoardState(s)
				step, ok := ApplyNakedSubset(s, size)
				if !ok || step.Kind != "elimination" || step.Number != 0 {
					t.Fatal("expected candidate-only deduction")
				}
				if s.Board != before.Board {
					t.Fatal("subset placed a digit")
				}
				if len(step.Visual.Sources) != size || len(step.Visual.Digits) != size || len(step.Visual.Units) != 1 || step.Visual.Units[0] != (VisualUnit{Kind: kind, Index: 0}) {
					t.Fatalf("incorrect evidence: %+v", step.Visual)
				}
				if len(step.Eliminations) != (9-size)*size {
					t.Fatalf("unexpected eliminations: %d", len(step.Eliminations))
				}
				var reported [9][9]uint16
				for _, e := range step.Eliminations {
					if reported[e.Row][e.Col]&(1<<e.Number) != 0 {
						t.Fatal("duplicate elimination")
					}
					reported[e.Row][e.Col] |= 1 << e.Number
				}
				for row := range 9 {
					for col := range 9 {
						cell := Cell{Row: row, Col: col}
						inside := kind == "row" && row == 0 || kind == "column" && col == 0 || kind == "box" && row < 3 && col < 3
						expected := before.Candidates[row][col]
						removed := uint16(0)
						if inside && !containsCell(sources, cell) {
							removed = expected & reserved
							expected &^= reserved
						}
						if s.Candidates[row][col] != expected || reported[row][col] != removed {
							t.Fatalf("incorrect elimination at %s", cellName(row, col))
						}
					}
				}
				after := CaptureBoardState(s)
				if _, ok := ApplyNakedSubset(s, size); ok || CaptureBoardState(s) != after {
					t.Fatal("reported a no-op subset")
				}
			})
		}
	}
}

func TestLockedNakedSubsetsAffectBothUnits(t *testing.T) {
	for size := 2; size <= 3; size++ {
		s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		digits := maskOf(2, 7)
		if size == 3 {
			digits = maskOf(2, 5, 7)
		}
		for col := range size {
			s.Candidates[0][col] = digits
		}
		step, ok := ApplyNakedSubset(s, size)
		if !ok || len(step.Visual.Units) != 2 {
			t.Fatal("locked subset must show the row and box")
		}
		seen := map[CandidateElimination]bool{}
		for _, e := range step.Eliminations {
			if seen[e] {
				t.Fatal("row/box overlap reported twice")
			}
			seen[e] = true
			if e.Row != 0 && !(e.Row < 3 && e.Col < 3) {
				t.Fatal("eliminated outside shared units")
			}
		}
		for row := range 9 {
			for col := range 9 {
				expected := AllCandidates
				if row == 0 && col < size {
					expected = digits
				} else if row == 0 || (row < 3 && col < 3) {
					expected &^= digits
				}
				if s.Candidates[row][col] != expected {
					t.Fatal("incorrect locked-subset candidates")
				}
			}
		}
	}
}

func TestNakedSubsetsRejectWrongUnionsAndSizes(t *testing.T) {
	cases := []struct {
		size  int
		masks []uint16
	}{
		{2, []uint16{maskOf(2, 5), maskOf(5, 8)}},
		{3, []uint16{maskOf(1, 2), maskOf(2, 3), maskOf(3, 4)}},
		{3, []uint16{maskOf(2, 7), maskOf(2, 7), maskOf(2, 7)}},
		{4, []uint16{maskOf(1, 2), maskOf(2, 3), maskOf(3, 4), maskOf(4, 5)}},
		{0, nil}, {1, nil}, {5, nil},
	}
	for i, tt := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
			for col, mask := range tt.masks {
				s.Candidates[0][col] = mask
			}
			before := CaptureBoardState(s)
			if _, ok := ApplyNakedSubset(s, tt.size); ok || CaptureBoardState(s) != before {
				t.Fatal("invalid subset changed candidates")
			}
		})
	}
}

func TestNakedSubsetIgnoresFilledCells(t *testing.T) {
	s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
	s.Candidates[0][0] = maskOf(2, 7)
	s.Candidates[0][4] = maskOf(2, 7)
	s.Board[0][8] = 1
	before := s.Candidates[0][8]
	step, ok := ApplyNakedSubset(s, 2)
	if !ok || s.Candidates[0][8] != before {
		t.Fatal("filled cell was modified")
	}
	for _, e := range step.Eliminations {
		if e.Row == 0 && e.Col == 8 {
			t.Fatal("filled cell reported as an elimination")
		}
	}
}

func TestSolveStepIncludesNakedSubsets(t *testing.T) {
	patterns := [][]uint16{
		{maskOf(2, 7), maskOf(2, 7)},
		{maskOf(2, 5), maskOf(5, 8), maskOf(2, 8)},
		{maskOf(1, 4), maskOf(4, 6), maskOf(6, 9), maskOf(1, 9)},
	}
	names := []string{"Naked Pair", "Naked Triple", "Naked Quadruple"}
	for i, pattern := range patterns {
		s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		for j, mask := range pattern {
			s.Candidates[0][[]int{0, 4, 8, 2}[j]] = mask
		}
		step, ok := s.SolveStep()
		if !ok || step.Technique != names[i] || step.Kind != "elimination" {
			t.Fatalf("expected %s, got %+v", names[i], step)
		}
	}
}
