package solver

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

type uniquenessFixture struct {
	Technique    string
	Puzzle       string
	Board        [9][9]int8
	Candidates   [9][9]uint16
	Eliminations []CandidateElimination
	Number       int
}

func uniquenessFixtures(t *testing.T) []uniquenessFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/uniqueness_steps.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []uniquenessFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 3 {
		t.Fatal("expected UR1, UR2, and BUG+1 reference states")
	}
	return fixtures
}

func applyUniquenessFixture(s *Sudoku, name string) (Step, bool) {
	switch name {
	case "Unique Rectangle Type 1":
		return ApplyUniqueRectangle(s, 1)
	case "Unique Rectangle Type 2":
		return ApplyUniqueRectangle(s, 2)
	default:
		return ApplyBUGPlusOne(s)
	}
}

func TestUniquenessMethodsOnVerifiedReferenceStates(t *testing.T) {
	for _, fixture := range uniquenessFixtures(t) {
		t.Run(fixture.Technique, func(t *testing.T) {
			s, err := NewSudoku(Puzzle{Puzzle: fixture.Puzzle})
			if err != nil {
				t.Fatal(err)
			}
			s.Board, s.Candidates = fixture.Board, fixture.Candidates
			before := CaptureBoardState(s)
			step, ok := applyUniquenessFixture(s, fixture.Technique)
			if !ok || step.Technique != fixture.Technique || !s.HasUniqueSolution() {
				t.Fatal("verified uniqueness step missing")
			}
			if !reflect.DeepEqual(step.Eliminations, fixture.Eliminations) || step.Number != fixture.Number {
				t.Fatal("reference step changed")
			}
			if fixture.Number == 0 {
				assertPatternStep(t, before, s, step, fixture.Technique)
			} else {
				before.Restore(s)
				placement, ok := s.SolveStep()
				if !ok || placement.Technique != "BUG+1" || placement.Kind != "placement" || len(placement.Eliminations) != 0 {
					t.Fatal("BUG+1 must record placement separately")
				}
				cleanup, ok := s.SolveStep()
				if !ok || cleanup.Technique != "Peer Elimination" {
					t.Fatal("BUG+1 did not get its separate cleanup")
				}
			}
		})
	}
}

func TestUniquenessMethodsRejectNonUniqueOriginalClues(t *testing.T) {
	for _, fixture := range uniquenessFixtures(t) {
		for _, original := range []string{strings.Repeat(".", 81), "531070000600195000098000060800060003400803001700020006060000280000419005000080079"} {
			s, err := NewSudoku(Puzzle{Puzzle: original})
			if err != nil {
				t.Fatal(err)
			}
			// Even if later placements resemble a unique puzzle, the original
			// submitted clues must independently pass the uniqueness gate.
			s.Board, s.Candidates = fixture.Board, fixture.Candidates
			before := CaptureBoardState(s)
			if _, ok := applyUniquenessFixture(s, fixture.Technique); ok || CaptureBoardState(s) != before {
				t.Fatal("uniqueness technique accepted multiple or zero solutions")
			}
		}
	}
}

func TestBUGPlusOneChecksCandidateDistribution(t *testing.T) {
	for _, fixture := range uniquenessFixtures(t) {
		if fixture.Technique != "BUG+1" {
			continue
		}
		s, _ := NewSudoku(Puzzle{Puzzle: fixture.Puzzle})
		s.Board, s.Candidates = fixture.Board, fixture.Candidates
		changed := false
		for row := range 9 {
			for col := range 9 {
				if changed || s.Board[row][col] != 0 || candidateCount(s.Candidates[row][col]) != 2 {
					continue
				}
				mask := s.Candidates[row][col]
				for number := 1; number <= 9; number++ {
					if !hasCandidate(mask, number) {
						s.Candidates[row][col] = removeCandidate(mask, candidateList(mask)[0]) | 1<<number
						changed = true
						break
					}
				}
			}
		}
		before := CaptureBoardState(s)
		if _, ok := ApplyBUGPlusOne(s); ok || CaptureBoardState(s) != before {
			t.Fatal("one trivalue cell alone was incorrectly considered BUG+1")
		}
	}
}

func TestUniqueRectangleShapeAndCandidateRequirements(t *testing.T) {
	for _, tt := range []struct {
		cells []Cell
		masks []uint16
		kind  int
	}{
		{[]Cell{{0, 0}, {0, 1}, {1, 0}, {1, 1}}, []uint16{maskOf(1, 2), maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3)}, 1},    // one box
		{[]Cell{{0, 0}, {0, 4}, {4, 0}, {4, 4}}, []uint16{maskOf(1, 2), maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3)}, 1},    // four boxes
		{[]Cell{{0, 0}, {0, 4}, {1, 0}, {1, 4}}, []uint16{maskOf(1, 2), maskOf(1, 2), maskOf(1, 2), maskOf(1, 3)}, 1},       // missing pair digit
		{[]Cell{{0, 0}, {0, 4}, {1, 0}, {1, 4}}, []uint16{maskOf(1, 2, 3), maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3)}, 2}, // diagonal roofs
		{[]Cell{{0, 0}, {0, 4}, {1, 0}, {1, 4}}, []uint16{maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3), maskOf(1, 2, 4)}, 2}, // different extras
	} {
		s := patternSudoku()
		// Isolate rectangle detection under an already verified uniqueness premise.
		s.initialKnown, s.uniqueness = true, 1
		for i, cell := range tt.cells {
			s.Candidates[cell.Row][cell.Col] = tt.masks[i]
		}
		before := CaptureBoardState(s)
		if _, ok := ApplyUniqueRectangle(s, tt.kind); ok || CaptureBoardState(s) != before {
			t.Fatal("invalid rectangle was accepted")
		}
	}
}
