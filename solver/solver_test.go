package solver

import (
	"strings"
	"testing"
)

const solvedGrid = "534678912672195348198342567859761423426853791713924856961537284287419635345286179"

func TestNewSudokuValidation(t *testing.T) {
	for _, puzzle := range []string{"", strings.Repeat(".", 80), "x" + strings.Repeat(".", 80), "11" + strings.Repeat(".", 79), "1" + strings.Repeat(".", 8) + "1" + strings.Repeat(".", 71), "1" + strings.Repeat(".", 9) + "1" + strings.Repeat(".", 70)} {
		if _, err := NewSudoku(Puzzle{Puzzle: puzzle}); err == nil {
			t.Errorf("accepted invalid puzzle %q", puzzle)
		}
	}
	s, err := NewSudoku(Puzzle{Puzzle: strings.Repeat("0", 81)})
	if err != nil {
		t.Fatal(err)
	}
	if s.Candidates[0][0] != AllCandidates {
		t.Fatal("empty board should allow all digits")
	}
}

func TestSolveHistoryAndAutomaticAgree(t *testing.T) {
	puzzle := Puzzle{Puzzle: "530070000600195000098000060800060003400803001700020006060000280000419005000080079"}
	s, err := NewSudoku(puzzle)
	if err != nil {
		t.Fatal(err)
	}
	initial := CaptureBoardState(s)
	history := s.Solve()
	if !s.IsSolved() {
		t.Fatal("expected solved puzzle")
	}
	if history.Entries[0].State != initial || history.Entries[0].HasStep {
		t.Fatal("initial state changed")
	}
	for i, entry := range history.Entries[1:] {
		if !entry.HasStep || entry.State == history.Entries[i].State {
			t.Fatal("step must change state")
		}
	}
	automatic, _ := NewSudoku(puzzle)
	automatic.SolveAutomatically()
	if CaptureBoardState(automatic) != CaptureBoardState(s) {
		t.Fatal("automatic and recorded solves disagree")
	}
	initial.Restore(s)
	if CaptureBoardState(s) != initial {
		t.Fatal("restore lost candidates or board values")
	}
}

func TestHistoryOwnsStepSlices(t *testing.T) {
	s := &Sudoku{}
	h := NewSolverHistory(s)
	step := Step{Details: []string{"original"}, Eliminations: []CandidateElimination{{Number: 1}}}
	h.RecordStep(s, step)
	step.Details[0] = "changed"
	step.Eliminations[0].Number = 2
	if h.Entries[1].Step.Details[0] != "original" || h.Entries[1].Step.Eliminations[0].Number != 1 {
		t.Fatal("history shares mutable step data")
	}
}

func TestSingleTechniquesApplyChanges(t *testing.T) {
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyFullHouse, ApplyHiddenSingle, ApplyNakedSingle} {
		s, err := NewSudoku(Puzzle{Puzzle: "." + solvedGrid[1:]})
		if err != nil {
			t.Fatal(err)
		}
		step, ok := apply(s)
		if !ok || step.Row != 0 || step.Col != 0 || step.Number != 5 || !s.IsSolved() {
			t.Fatal("technique did not apply its returned placement")
		}
	}
}

func TestStuckIsNotSolved(t *testing.T) {
	s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
	if h := s.Solve(); h.Len() != 1 || s.IsSolved() {
		t.Fatal("empty puzzle must stop unsolved without fabricated steps")
	}
}

func TestCandidateEliminations(t *testing.T) {
	t.Run("pointing", func(t *testing.T) {
		s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		for row := 1; row < 3; row++ {
			for col := 0; col < 3; col++ {
				s.Candidates[row][col] = removeCandidate(s.Candidates[row][col], 5)
			}
		}
		step, ok := ApplyPointing(s)
		if !ok || len(step.Eliminations) != 6 {
			t.Fatal("expected six eliminations outside the box")
		}
		for _, e := range step.Eliminations {
			if e.Row != 0 || e.Col < 3 || e.Number != 5 || hasCandidate(s.Candidates[e.Row][e.Col], 5) {
				t.Fatal("incorrect pointing elimination")
			}
		}
	})
	t.Run("hidden pair", func(t *testing.T) {
		s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		for col := 2; col < 9; col++ {
			s.Candidates[0][col] = removeCandidate(removeCandidate(s.Candidates[0][col], 1), 2)
		}
		step, ok := ApplyHiddenSubset(s, 2)
		if !ok || len(step.Eliminations) != 14 {
			t.Fatal("expected other digits removed from pair")
		}
		for col := 0; col < 2; col++ {
			if s.Candidates[0][col] != (1<<1 | 1<<2) {
				t.Fatal("hidden pair retained extra candidates")
			}
		}
	})
}
