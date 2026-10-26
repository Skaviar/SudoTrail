package puzzlebank

import (
	"bytes"
	"testing"

	"github.com/s-threepio/godoku/solver"
)

func TestEmbeddedCatalog(t *testing.T) {
	if len(puzzles) != 1000 {
		t.Fatalf("expected 1000 puzzles, got %d", len(puzzles))
	}
	seen := map[string]bool{}
	for _, p := range puzzles {
		if seen[p.Puzzle] {
			t.Fatalf("duplicate puzzle %s", p.ID)
		}
		seen[p.Puzzle] = true
		if issues := solver.ValidatePuzzle(p.Puzzle); len(issues) > 0 {
			t.Fatalf("invalid puzzle %s: %+v", p.ID, issues)
		}
	}
	if bytes.Contains(catalog, []byte(`"solution"`)) {
		t.Fatal("do not ship database solutions")
	}
	for range 100 {
		p := Random()
		if !seen[p.Puzzle] {
			t.Fatal("random result outside embedded bank")
		}
	}
}

func TestTeachingAndAutomaticSolvesAgree(t *testing.T) {
	// Exercise different database puzzles, including ones that stop unsolved.
	for _, p := range puzzles[:100] {
		teaching, err := solver.NewSudoku(solver.Puzzle{Puzzle: p.Puzzle})
		if err != nil {
			t.Fatal(err)
		}
		automatic, err := solver.NewSudoku(solver.Puzzle{Puzzle: p.Puzzle})
		if err != nil {
			t.Fatal(err)
		}
		for {
			step, ok := teaching.SolveStep()
			if !ok {
				break
			}
			if step.Number > 0 && len(step.Eliminations) > 0 {
				t.Fatal("placement and elimination were merged")
			}
		}
		automatic.SolveAutomatically()
		if teaching.Board != automatic.Board || teaching.Candidates != automatic.Candidates {
			t.Fatalf("teaching and benchmark mode disagree for puzzle %s", p.ID)
		}
	}
}
