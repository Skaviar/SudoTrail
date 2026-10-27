package solver

import "testing"

func patternSudoku() *Sudoku {
	s := &Sudoku{}
	for row := range 9 {
		for col := range 9 {
			s.Candidates[row][col] = AllCandidates
		}
	}
	return s
}

// Check the observable change and the truth of every displayed relationship.
func assertPatternStep(t *testing.T, before BoardState, s *Sudoku, step Step, technique string) {
	t.Helper()
	if step.Technique != technique || step.Kind != "elimination" || step.Number != 0 || len(step.Eliminations) == 0 || step.Visual == nil {
		t.Fatalf("expected %s elimination, got %+v", technique, step)
	}
	if s.Board != before.Board {
		t.Fatal("elimination placed a number")
	}
	expected := before.Candidates
	seen := map[CandidateElimination]bool{}
	for _, e := range step.Eliminations {
		if seen[e] || before.Board[e.Row][e.Col] != 0 || !hasCandidate(expected[e.Row][e.Col], e.Number) {
			t.Fatalf("duplicate, filled, or missing candidate: %+v", e)
		}
		seen[e] = true
		expected[e.Row][e.Col] = removeCandidate(expected[e.Row][e.Col], e.Number)
		if expected[e.Row][e.Col] == 0 {
			t.Fatal("deduction left an empty candidate set")
		}
	}
	if s.Candidates != expected {
		t.Fatal("candidate changes differ from the reported eliminations")
	}
	for _, source := range step.Visual.Sources {
		if before.Board[source.Row][source.Col] != 0 || s.Candidates[source.Row][source.Col] != before.Candidates[source.Row][source.Col] {
			t.Fatal("pattern source was filled or altered")
		}
	}
	units := allUnits()
	for _, link := range step.Visual.CandidateLinks {
		if !cellsSeeEachOther(link.From, link.To) || !hasCandidate(before.Candidates[link.From.Row][link.From.Col], link.Number) || !hasCandidate(before.Candidates[link.To.Row][link.To.Col], link.Number) {
			t.Fatalf("invented candidate connection: %+v", link)
		}
		index := link.Unit.Index
		if link.Unit.Kind == "column" {
			index += 9
		} else if link.Unit.Kind == "box" {
			index += 18
		}
		if !containsCell(units[index].cells, link.From) || !containsCell(units[index].cells, link.To) {
			t.Fatalf("link names the wrong shared unit: %+v", link)
		}
		if link.Kind == "strong" {
			copy := &Sudoku{Board: before.Board, Candidates: before.Candidates}
			if len(candidateCells(copy, units[index], link.Number)) != 2 {
				t.Fatalf("strong link has additional possible locations: %+v", link)
			}
		}
	}
}
