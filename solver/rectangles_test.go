package solver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func rectanglePattern(masks [4]uint16) *Sudoku {
	s := patternSudoku()
	// These constructed examples isolate the rule under an explicit unique
	// puzzle premise. Separate real-puzzle tests verify the uniqueness check.
	s.initialKnown, s.uniqueness = true, 1
	for i, c := range []Cell{{0, 0}, {0, 4}, {1, 0}, {1, 4}} {
		s.Candidates[c.Row][c.Col] = masks[i]
	}
	return s
}

func TestUniqueRectangleTypeFive(t *testing.T) {
	for _, masks := range [][4]uint16{
		{maskOf(1, 2, 3), maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3)},    // diagonal roofs
		{maskOf(1, 2, 3), maskOf(1, 2, 3), maskOf(1, 2), maskOf(1, 2, 3)}, // three roofs
	} {
		s := rectanglePattern(masks)
		before := CaptureBoardState(s)
		step, ok := ApplyUniqueRectangle(s, 5)
		if !ok || step.Technique != "Unique Rectangle Type 5" {
			t.Fatalf("type 5 not recognized: %+v", step)
		}
		assertPatternStep(t, before, s, step, step.Technique)
		for _, e := range step.Eliminations {
			if e.Number != 3 {
				t.Fatal("removed a rectangle digit instead of the extra digit")
			}
		}
		// Neither arbitrary candidate removals nor a promising rectangle
		// may replace verification of the original puzzle's uniqueness.
		before.Restore(s)
		s.uniqueness = 0
		if _, ok := ApplyAdvancedRectangles(s); ok || CaptureBoardState(s) != before {
			t.Fatal("type 5 bypassed uniqueness gate")
		}
	}
}

func TestRectangleEntryPointsAndUniquenessGateOnReferenceStates(t *testing.T) {
	data, err := os.ReadFile("testdata/advanced_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct{ Technique, Puzzle string }
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	methods := map[string]func(*Sudoku) (Step, bool){
		"Unique Rectangle Type 3":                      func(s *Sudoku) (Step, bool) { return ApplyUniqueRectangle(s, 3) },
		"Unique Rectangle Type 4":                      func(s *Sudoku) (Step, bool) { return ApplyUniqueRectangle(s, 4) },
		"Unique Rectangle Type 6":                      func(s *Sudoku) (Step, bool) { return ApplyUniqueRectangle(s, 6) },
		"Hidden Rectangle":                             ApplyHiddenRectangle,
		"Avoidable Rectangle Type 1":                   ApplyAvoidableRectangle,
		"Avoidable Rectangle Type 2":                   ApplyAvoidableRectangle,
		"Unique Rectangle Type 1 (Missing Candidates)": ApplyMissingCandidateRectangles,
	}
	for _, fixture := range fixtures {
		apply := methods[fixture.Technique]
		if apply == nil {
			continue
		}
		s, _ := NewSudoku(Puzzle{Puzzle: fixture.Puzzle})
		history := s.Solve()
		for i, entry := range history.Entries {
			if entry.Step.Technique != fixture.Technique {
				continue
			}
			before := history.Entries[i-1].State
			before.Restore(s)
			if step, ok := apply(s); !ok || step.Technique != fixture.Technique {
				t.Fatalf("named entry point failed: %s", fixture.Technique)
			}
			// Later placements/candidates must not make a blank original
			// puzzle appear unique, even for an Avoidable Rectangle.
			before.Restore(s)
			s.initialBoard = [9][9]int8{}
			s.uniqueness = 0
			unchanged := CaptureBoardState(s)
			if _, ok := apply(s); ok || CaptureBoardState(s) != unchanged {
				t.Fatalf("original-clue uniqueness gate bypassed: %s", fixture.Technique)
			}
			delete(methods, fixture.Technique)
			break
		}
	}
	if len(methods) > 0 {
		t.Fatalf("missing reference coverage for %v", methods)
	}
}

func TestMissingRectangleCandidatesMustBeAllowedByGivens(t *testing.T) {
	s := rectanglePattern([4]uint16{maskOf(1), maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3)})
	before := CaptureBoardState(s)
	step, ok := ApplyMissingCandidateRectangles(s)
	if !ok || step.Technique != "Unique Rectangle Type 1 (Missing Candidates)" {
		t.Fatalf("missing-candidate UR1 not found: %+v", step)
	}
	assertPatternStep(t, before, s, step, step.Technique)
	before.Restore(s)
	s.initialBoard[0][8] = 2 // A given blocks the missing 2 in the rectangle.
	original := &Sudoku{Board: s.initialBoard}
	original.initializeCandidates()
	if rectanglePairAllowed([]Cell{{0, 0}, {0, 4}, {1, 0}, {1, 4}}, original.Candidates, maskOf(1, 2)) {
		t.Fatal("restored a candidate blocked by a given")
	}
}

func TestNewUniquenessMethodsUseOriginalClues(t *testing.T) {
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyAdvancedRectangles, ApplyMissingCandidateRectangles, ApplyAvoidableRectangle} {
		for _, original := range []string{strings.Repeat(".", 81), "531070000600195000098000060800060003400803001700020006060000280000419005000080079"} {
			s, _ := NewSudoku(Puzzle{Puzzle: original})
			pattern := rectanglePattern([4]uint16{maskOf(1, 2, 3), maskOf(1, 2), maskOf(1, 2), maskOf(1, 2, 3)})
			s.Board, s.Candidates = pattern.Board, pattern.Candidates
			before := CaptureBoardState(s)
			if _, ok := apply(s); ok || CaptureBoardState(s) != before {
				t.Fatal("uniqueness deduction accepted non-unique/unsatisfiable clues")
			}
		}
	}
}
