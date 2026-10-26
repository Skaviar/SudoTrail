package solver

import (
	"strings"
	"testing"
)

const teachingPuzzle = "530070000600195000098000060800060003400803001700020006060000280000419005000080079"

func TestPlacementAndPropagationAreSeparate(t *testing.T) {
	s, err := NewSudoku(Puzzle{Puzzle: teachingPuzzle})
	if err != nil {
		t.Fatal(err)
	}
	before := CaptureBoardState(s)
	placement, ok := s.SolveStep()
	if !ok || placement.Kind != "placement" || len(placement.Eliminations) != 0 {
		t.Fatal("first step must be a placement alone")
	}
	if s.pending.Count == 0 {
		t.Fatal("fixture must have pending peer eliminations")
	}
	after := CaptureBoardState(s)
	for r := range 9 {
		for c := range 9 {
			if r == placement.Row && c == placement.Col {
				if int(s.Board[r][c]) != placement.Number || s.Candidates[r][c] != 0 {
					t.Fatal("placement not applied")
				}
			} else if s.Board[r][c] != before.Board[r][c] || s.Candidates[r][c] != before.Candidates[r][c] {
				t.Fatal("placement changed a peer before its elimination step")
			}
		}
	}
	elimination, ok := s.SolveStep()
	if !ok || elimination.Technique != "Peer Elimination" || elimination.Kind != "elimination" || elimination.Number != 0 {
		t.Fatal("expected explicit cleanup immediately after placement")
	}
	if s.Board != after.Board {
		t.Fatal("cleanup must not place anything")
	}
	expected := after.Candidates
	for _, e := range elimination.Eliminations {
		if e.Number != placement.Number || !hasCandidate(expected[e.Row][e.Col], e.Number) {
			t.Fatal("reported a candidate that was not present")
		}
		expected[e.Row][e.Col] = removeCandidate(expected[e.Row][e.Col], e.Number)
	}
	if s.Candidates != expected {
		t.Fatal("cleanup differs from its recorded eliminations")
	}
	for _, link := range elimination.Visual.Links {
		if link.From != (Cell{Row: placement.Row, Col: placement.Col}) || int(after.Board[link.From.Row][link.From.Col]) != link.Number {
			t.Fatal("cleanup link has wrong source")
		}
	}
	// Restoring the intermediate placement must also restore its pending cleanup.
	after.Restore(s)
	replay, ok := s.SolveStep()
	if !ok || replay.Technique != elimination.Technique || s.Candidates != expected {
		t.Fatal("restoring an intermediate state lost pending cleanup")
	}
	after.Restore(s)
	s.SolveAutomatically()
	if !s.IsSolved() {
		t.Fatal("automatic solving could not resume from a pending cleanup")
	}
}

func TestBlockingLinksUseTheActualNumberAndHouse(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source Cell
		kind   string
	}{
		{"row", Cell{Row: 4, Col: 8}, "row"},
		{"column", Cell{Row: 8, Col: 4}, "column"},
		{"box", Cell{Row: 3, Col: 3}, "box"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sudoku{}
			s.Board[tt.source.Row][tt.source.Col] = 7
			link, ok := blockingNumber(s, Cell{Row: 4, Col: 4}, 7)
			if !ok || link.From != tt.source || link.Number != 7 || link.Unit.Kind != tt.kind {
				t.Fatalf("wrong blocking evidence: %+v", link)
			}
		})
	}
	s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
	s.Candidates[0][0] = removeCandidate(s.Candidates[0][0], 7)
	v := &StepVisual{}
	addRestriction(s, v, Cell{Row: 0, Col: 0}, 7)
	if len(v.Restrictions) != 1 || len(v.Links) != 0 || len(v.Sources) != 0 {
		t.Fatal("an earlier candidate elimination must not invent a placed source")
	}
}

func TestEveryPlacementLinkExplainsAnExistingRestriction(t *testing.T) {
	s, _ := NewSudoku(Puzzle{Puzzle: teachingPuzzle})
	linksChecked := 0
	for {
		before := CaptureBoardState(s)
		step, ok := s.SolveStep()
		if !ok {
			break
		}
		if step.Kind != "placement" {
			continue
		}
		for _, link := range step.Visual.Links {
			linksChecked++
			if int(before.Board[link.From.Row][link.From.Col]) != link.Number {
				t.Fatal("placement guide starts at a cell without its digit")
			}
			if before.Board[link.To.Row][link.To.Col] != 0 || hasCandidate(before.Candidates[link.To.Row][link.To.Col], link.Number) {
				t.Fatal("placement guide does not describe an existing restriction")
			}
			if sharedUnit(link.From, link.To) != link.Unit {
				t.Fatal("wrong shared unit in placement guide")
			}
		}
	}
	if linksChecked == 0 {
		t.Fatal("fixture did not exercise any placement explanation links")
	}
}
