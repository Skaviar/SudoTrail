package solver

import (
	"strings"
	"testing"
)

func TestEliminationVisualEvidence(t *testing.T) {
	for _, column := range []bool{false, true} {
		s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		for r := range 3 {
			for c := range 3 {
				if (!column && r != 0) || (column && c != 0) {
					s.Candidates[r][c] = removeCandidate(s.Candidates[r][c], 5)
				}
			}
		}
		step, ok := ApplyPointing(s)
		if !ok || step.Visual == nil || len(step.Visual.Sources) != 3 || len(step.Visual.Units) != 2 || step.Visual.Digits[0] != 5 {
			t.Fatal("missing pointing evidence")
		}
		kind := "row"
		if column {
			kind = "column"
		}
		if step.Visual.Units[1].Kind != kind {
			t.Fatal("incorrect axis")
		}
		for _, cell := range step.Visual.Sources {
			if !hasCandidate(s.Candidates[cell.Row][cell.Col], 5) {
				t.Fatal("source digit must remain")
			}
		}
	}
	s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
	for col := 2; col < 9; col++ {
		s.Candidates[0][col] = removeCandidate(removeCandidate(s.Candidates[0][col], 1), 2)
	}
	step, ok := ApplyHiddenSubset(s, 2)
	if !ok || len(step.Visual.Sources) != 2 || len(step.Visual.Digits) != 2 || step.Visual.Units[0].Kind != "row" {
		t.Fatal("missing hidden-pair evidence")
	}
}

func TestHistoryOwnsVisualSlices(t *testing.T) {
	s := &Sudoku{}
	h := NewSolverHistory(s)
	step := Step{Visual: &StepVisual{Sources: []Cell{{Row: 1, Col: 2}}, Digits: []int{5}, Units: []VisualUnit{{Kind: "row", Index: 1}}, Links: []VisualLink{{Number: 5}}, CandidateLinks: []CandidateLink{{Number: 5}}, Restrictions: []CandidateElimination{{Number: 5}}}}
	h.RecordStep(s, step)
	step.Visual.Sources[0].Row = 8
	step.Visual.Digits[0] = 9
	step.Visual.Units[0].Index = 8
	step.Visual.Links[0].Number = 9
	step.Visual.Restrictions[0].Number = 9
	step.Visual.CandidateLinks[0].Number = 9
	visual := h.Entries[1].Step.Visual
	if visual.Sources[0].Row != 1 || visual.Digits[0] != 5 || visual.Units[0].Index != 1 || visual.Links[0].Number != 5 || visual.Restrictions[0].Number != 5 || visual.CandidateLinks[0].Number != 5 {
		t.Fatal("visual evidence was mutated after recording")
	}
}
