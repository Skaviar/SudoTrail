package puzzlebank

import (
	"bytes"
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/s-threepio/godoku/solver"
)

func TestEmbeddedCatalog(t *testing.T) {
	if len(puzzles)%4 != 0 || len(puzzles) < 800 {
		t.Fatalf("expected a balanced bank with at least 200 puzzles per level, got %d", len(puzzles))
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, p := range puzzles {
		if seen[p.Puzzle] {
			t.Fatalf("duplicate puzzle %s", p.ID)
		}
		seen[p.Puzzle] = true
		counts[p.Difficulty]++
		if len(p.Techniques) == 0 || !sort.StringsAreSorted(p.Techniques) {
			t.Fatal("missing sorted practice metadata")
		}
		for _, technique := range p.Techniques {
			if _, _, known := solver.TechniqueDifficulty(technique); !known || technique == "Peer Elimination" {
				t.Fatalf("invalid practice technique %s", technique)
			}
		}
		if issues := solver.ValidatePuzzle(p.Puzzle); len(issues) > 0 {
			t.Fatalf("invalid puzzle %s: %+v", p.ID, issues)
		}
	}
	for _, level := range Levels {
		if counts[level] != len(puzzles)/4 {
			t.Fatalf("bank is not balanced: %v", counts)
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
	// Exercise 25 catalog puzzles from each difficulty tier.
	for i := range 100 {
		p := puzzles[i/25*(len(puzzles)/4)+i%25]
		teaching, err := solver.NewSudoku(solver.Puzzle{Puzzle: p.Puzzle})
		if err != nil {
			t.Fatal(err)
		}
		automatic, err := solver.NewSudoku(solver.Puzzle{Puzzle: p.Puzzle})
		if err != nil {
			t.Fatal(err)
		}
		for {
			step, ok := teaching.SolvePracticeStep()
			if !ok {
				break
			}
			if step.Number > 0 && len(step.Eliminations) > 0 {
				t.Fatal("placement and elimination were merged")
			}
		}
		rating, err := automatic.SolveRated(context.Background())
		if err != nil || rating.Status != "solved" || string(rating.Difficulty) != p.Difficulty {
			t.Fatalf("wrong catalog difficulty: %s %+v", p.ID, rating)
		}
		var names []string
		for name := range rating.TechniqueCounts {
			names = append(names, name)
		}
		sort.Strings(names)
		if !reflect.DeepEqual(names, p.Techniques) {
			t.Fatalf("practice metadata differs from hints for %s", p.ID)
		}
		if teaching.Board != automatic.Board || teaching.Candidates != automatic.Candidates {
			t.Fatalf("teaching and benchmark mode disagree for puzzle %s", p.ID)
		}
	}
}

func TestDifficultyAndTechniqueSelection(t *testing.T) {
	for _, level := range Catalog() {
		if level.Count != len(puzzles)/4 || len(level.Techniques) == 0 {
			t.Fatal("invalid catalog summary")
		}
		for _, technique := range level.Techniques {
			p, err := Select(Selection{Difficulty: level.Difficulty, Technique: technique.Name})
			if err != nil || p.Difficulty != level.Difficulty {
				t.Fatalf("filter failed: %v", err)
			}
			found := false
			for _, name := range p.Techniques {
				found = found || name == technique.Name
			}
			if !found {
				t.Fatal("returned a puzzle without requested technique")
			}
		}
		first, _ := Select(Selection{Difficulty: level.Difficulty})
		for range 20 {
			next, err := Select(Selection{Difficulty: level.Difficulty, Exclude: first.ID})
			if err != nil || next.ID == first.ID {
				t.Fatal("repeated excluded puzzle")
			}
		}
	}
	for _, selection := range []Selection{{Difficulty: "extreme"}, {Difficulty: "easy", Technique: "Forcing Net"}, {Technique: "not a technique"}} {
		if _, err := Select(selection); err == nil {
			t.Fatal("invalid filters silently fell back to random")
		}
	}
}
