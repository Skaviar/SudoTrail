package solver

import "fmt"

// A cell has at most 20 distinct peers. Keeping this update as a value also
// makes history snapshots independent and safe to restore between both phases.
type candidateUpdate struct {
	Source       Cell
	Number       int
	Eliminations [20]CandidateElimination
	Count        int
}

func (s *Sudoku) peerUpdate(row, col, number int) candidateUpdate {
	update := candidateUpdate{Source: Cell{Row: row, Col: col}, Number: number}
	for r := range 9 {
		for c := range 9 {
			if s.Board[r][c] != 0 || !hasCandidate(s.Candidates[r][c], number) {
				continue
			}
			if r == row || c == col || (r/3 == row/3 && c/3 == col/3) {
				update.Eliminations[update.Count] = CandidateElimination{Row: r, Col: c, Number: number}
				update.Count++
			}
		}
	}
	if update.Count == 0 {
		return candidateUpdate{}
	}
	return update
}

func (s *Sudoku) applyPendingCandidates() (Step, bool) {
	update := s.pending
	if update.Count == 0 {
		return Step{}, false
	}
	s.pending = candidateUpdate{}
	eliminations := append([]CandidateElimination(nil), update.Eliminations[:update.Count]...)
	visual := &StepVisual{Sources: []Cell{update.Source}, Digits: []int{update.Number}}
	details := []string{fmt.Sprintf("%s now contains %d. Remove %d from its empty peers.", cellName(update.Source.Row, update.Source.Col), update.Number, update.Number)}
	for _, e := range eliminations {
		target := Cell{Row: e.Row, Col: e.Col}
		unit := sharedUnit(update.Source, target)
		s.Candidates[e.Row][e.Col] = removeCandidate(s.Candidates[e.Row][e.Col], e.Number)
		visual.Links = append(visual.Links, VisualLink{From: update.Source, To: target, Number: e.Number, Unit: unit})
		details = append(details, fmt.Sprintf("%s and %s share %s, so %s cannot contain %d.", cellName(update.Source.Row, update.Source.Col), cellName(e.Row, e.Col), unitLabel(unit), cellName(e.Row, e.Col), e.Number))
	}
	return Step{
		Kind: "elimination", Technique: "Peer Elimination",
		Explanation: fmt.Sprintf("Remove %d from peers of %s.", update.Number, cellName(update.Source.Row, update.Source.Col)),
		Details:     details, Eliminations: eliminations, Visual: visual,
	}, true
}
