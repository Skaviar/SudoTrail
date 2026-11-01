package solver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestRatingsOnReferencePuzzles(t *testing.T) {
	data, err := os.ReadFile("testdata/rated_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		ID, Puzzle, Solution, Difficulty, Score string
		Steps                                   string `json:"logical_steps"`
		Hardest                                 string `json:"hardest_technique"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		t.Run(f.Difficulty, func(t *testing.T) {
			s, _ := NewSudoku(Puzzle{Puzzle: f.Puzzle})
			rating, err := s.SolveRated(context.Background())
			score, _ := strconv.Atoi(f.Score)
			steps, _ := strconv.Atoi(f.Steps)
			if err != nil || rating.Status != "solved" || string(rating.Difficulty) != f.Difficulty || rating.Score != score || rating.LogicalSteps != steps || rating.HardestTechnique != f.Hardest {
				t.Fatalf("unexpected reference rating: %+v, %v", rating, err)
			}
			for i, ch := range []byte(f.Solution) {
				if s.Board[i/9][i%9] != int8(ch-'0') {
					t.Fatal("rating did not preserve reference solution")
				}
			}
			// Neither a supplied answer nor the old catalog rating may
			// affect the new grade or score.
			again, err := RatePuzzle(context.Background(), Puzzle{Puzzle: f.Puzzle, Solution: "wrong supplied answer", Difficulty: "wrong old rating"})
			if err != nil || !reflect.DeepEqual(rating, again) {
				t.Fatal("rating depends on input metadata")
			}
		})
	}
}

func TestRatedSolvePreservesEveryReferenceCandidate(t *testing.T) {
	data, err := os.ReadFile("testdata/advanced_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct{ Puzzle, Solution string }
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, f := range fixtures {
		if seen[f.Puzzle] {
			continue
		}
		seen[f.Puzzle] = true
		s, _ := NewSudoku(Puzzle{Puzzle: f.Puzzle})
		for {
			step, ok := s.applyRatingTechnique()
			if !ok {
				break
			}
			if _, _, known := TechniqueDifficulty(step.Technique); !known {
				t.Fatalf("unrated technique %s", step.Technique)
			}
			for i, ch := range []byte(f.Solution) {
				r, c, d := i/9, i%9, int(ch-'0')
				if value := s.Board[r][c]; value != 0 && int(value) != d || value == 0 && !hasCandidate(s.Candidates[r][c], d) {
					t.Fatalf("rating order lost a solution candidate with %s", step.Technique)
				}
			}
		}
		if !s.IsSolved() {
			t.Fatal("rating stalled on a reference solved by the available techniques")
		}
	}
}

func TestRatingExcludesAutomaticPeerCleanup(t *testing.T) {
	rating, err := RatePuzzle(context.Background(), Puzzle{Puzzle: teachingPuzzle})
	if err != nil || rating.Status != "solved" || rating.Difficulty != Easy {
		t.Fatalf("unexpected rating %+v %v", rating, err)
	}
	if rating.LogicalSteps != 51 || rating.TechniqueCounts["Peer Elimination"] != 0 {
		t.Fatal("automatic peer updates inflated the hint count")
	}
	count := 0
	for _, n := range rating.TechniqueCounts {
		count += n
	}
	if count != rating.LogicalSteps {
		t.Fatal("technique counts disagree with logical steps")
	}
}

func TestRatingTriesHardAlternativesBeforeFinnedFish(t *testing.T) {
	// Catalog puzzle 349 has a finned-fish shortcut in the teaching order,
	// but the rating order can finish it entirely with Hard-or-easier moves.
	const puzzle = "...8....9.....1..47..5..1..6.5....4..493.5..7.3.4...6..937......7.....51.....2..."
	teaching, _ := NewSudoku(Puzzle{Puzzle: puzzle})
	history := teaching.Solve()
	usedFinnedFish := false
	for _, entry := range history.Entries {
		usedFinnedFish = usedFinnedFish || entry.Step.Technique == "Finned X-Wing"
	}
	if !usedFinnedFish {
		t.Fatal("reference no longer exercises the ordering difference")
	}
	rating, err := RatePuzzle(context.Background(), Puzzle{Puzzle: puzzle})
	if err != nil || rating.Status != "solved" || rating.Difficulty != Hard {
		t.Fatalf("an available easier solve path was overrated: %+v %v", rating, err)
	}
	for technique := range rating.TechniqueCounts {
		if level, _, _ := TechniqueDifficulty(technique); level == Expert {
			t.Fatal("expert technique was used before exhausting Hard")
		}
	}
}

func TestUnsolvedAndInvalidPuzzlesRemainUngraded(t *testing.T) {
	for _, tt := range []struct{ puzzle, status string }{{strings.Repeat(".", 81), "unsolved"}, {"11" + strings.Repeat(".", 79), "invalid"}} {
		rating, err := RatePuzzle(context.Background(), Puzzle{Puzzle: tt.puzzle})
		if err != nil || rating.Status != tt.status || rating.Difficulty != "" || rating.LogicalSteps != 0 {
			t.Fatalf("incorrect ungraded status: %+v %v", rating, err)
		}
	}
	rating, err := RatePuzzle(context.Background(), Puzzle{Puzzle: solvedGrid})
	if err != nil || rating.Status != "solved" || rating.Difficulty != Easy || rating.Score != 0 || rating.HardestTechnique != "" {
		t.Fatal("a completed grid should need zero deductions")
	}
}

func TestRatingCancellationDoesNotChangePuzzle(t *testing.T) {
	s, _ := NewSudoku(Puzzle{Puzzle: teachingPuzzle})
	s.SolveStep()
	before := CaptureBoardState(s)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.SolveRated(ctx); !errors.Is(err, context.Canceled) || CaptureBoardState(s) != before {
		t.Fatal("canceled rating changed a pending teaching state")
	}
}

func TestRatingPolicyCoversTheTechniqueCatalog(t *testing.T) {
	data, err := os.ReadFile("testdata/advanced_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct{ Technique string }
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		if _, points, known := TechniqueDifficulty(f.Technique); !known || points <= 0 {
			t.Fatalf("missing rating for %s", f.Technique)
		}
	}
	for size := 2; size <= 7; size++ {
		for _, prefix := range []string{"Finned ", "Sashimi ", "Franken ", "Finned Franken ", "Mutant ", "Finned Mutant ", "Siamese ", "Siamese Finned Mutant "} {
			if level, points, known := TechniqueDifficulty(prefix + fishNames[size]); !known || level != Expert || points <= 0 {
				t.Fatal("missing expert fish rating")
			}
		}
	}
	if _, _, known := TechniqueDifficulty("A future unknown technique"); known {
		t.Fatal("unknown techniques must not be assigned a silent default")
	}
}
