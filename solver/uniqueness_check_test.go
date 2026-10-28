package solver

import (
	"strings"
	"testing"
)

func TestUniquenessCheckCountsSolutionsWithoutChangingPuzzle(t *testing.T) {
	for _, tt := range []struct {
		name, puzzle string
		want         int
	}{
		{"complete", solvedGrid, 1},
		{"unique", teachingPuzzle, 1},
		{"multiple", strings.Repeat(".", 81), 2},
		{"unsatisfiable", "531070000600195000098000060800060003400803001700020006060000280000419005000080079", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSudoku(Puzzle{Puzzle: tt.puzzle})
			if err != nil {
				t.Fatal(err)
			}
			before := CaptureBoardState(s)
			if got := countSolutions(s.Board); got != tt.want {
				t.Fatalf("got %d solutions, want %d", got, tt.want)
			}
			if s.HasUniqueSolution() != (tt.want == 1) {
				t.Fatal("incorrect uniqueness verdict")
			}
			if CaptureBoardState(s) != before {
				t.Fatal("uniqueness check changed the puzzle")
			}
		})
	}
	for _, value := range []int8{-1, 10, 3} {
		s, _ := NewSudoku(Puzzle{Puzzle: teachingPuzzle})
		s.Board[0][0] = value // 3 conflicts with the second clue in this row.
		if countSolutions(s.Board) != 0 {
			t.Fatal("uniqueness check accepted an invalid board")
		}
	}
}

func TestUniquenessCheckIgnoresLaterCandidateEliminations(t *testing.T) {
	s, _ := NewSudoku(Puzzle{Puzzle: teachingPuzzle})
	s.Candidates[0][2] = removeCandidate(s.Candidates[0][2], 4)
	before := CaptureBoardState(s)
	if !s.HasUniqueSolution() || CaptureBoardState(s) != before {
		t.Fatal("candidate changes affected the original clues' uniqueness")
	}
}

func TestUniquenessCheckDoesNotCompleteStalledPuzzle(t *testing.T) {
	const puzzle = "..61.4.9.35...9......25.........5..8......2...324...718...9.3...95...7...4.7.1..."
	s, err := NewSudoku(Puzzle{Puzzle: puzzle})
	if err != nil {
		t.Fatal(err)
	}
	before := CaptureBoardState(s)
	if !s.HasUniqueSolution() || CaptureBoardState(s) != before {
		t.Fatal("uniqueness check must only confirm the puzzle is unique")
	}
	s.Solve()
	if s.IsSolved() {
		t.Fatal("logical solving completed a puzzle that needs unsupported techniques")
	}
	stuck := CaptureBoardState(s)
	if !s.HasUniqueSolution() {
		t.Fatal("solving changed the original uniqueness verdict")
	}
	if _, ok := s.SolveStep(); ok {
		t.Fatal("stalled solver made a step after checking uniqueness")
	}
	s.SolveAutomatically()
	if CaptureBoardState(s) != stuck {
		t.Fatal("uniqueness check supplied a move to the stalled solver")
	}
}

func TestUniquenessUsesOriginalCluesAndSurvivesRestore(t *testing.T) {
	blank, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
	initial := CaptureBoardState(blank)
	if blank.HasUniqueSolution() {
		t.Fatal("empty puzzle is not unique")
	}
	complete, _ := NewSudoku(Puzzle{Puzzle: solvedGrid})
	if !complete.HasUniqueSolution() {
		t.Fatal("completed valid puzzle is unique")
	}
	blank.Board = complete.Board
	if blank.HasUniqueSolution() {
		t.Fatal("later placements changed the original puzzle's uniqueness")
	}
	initial.Restore(complete)
	if complete.HasUniqueSolution() {
		t.Fatal("restoring different clues retained a stale uniqueness cache")
	}
}
