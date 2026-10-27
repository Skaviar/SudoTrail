package solver

import "fmt"

// These helpers are shared by fish, wings, and single-digit patterns.
func cellHasCandidate(s *Sudoku, cell Cell, number int) bool {
	return s.Board[cell.Row][cell.Col] == 0 && hasCandidate(s.Candidates[cell.Row][cell.Col], number)
}

func candidateCells(s *Sudoku, u unit, number int) []Cell {
	var cells []Cell
	for _, cell := range u.cells {
		if int(s.Board[cell.Row][cell.Col]) == number {
			return nil // This unit already has its digit.
		}
		if cellHasCandidate(s, cell, number) {
			cells = append(cells, cell)
		}
	}
	return cells
}

// A conjugate pair is the only two possible locations of a digit in one unit.
type conjugatePair struct {
	a, b Cell
	unit VisualUnit
}

func conjugatePairs(s *Sudoku, number int) []conjugatePair {
	var pairs []conjugatePair
	for _, u := range allUnits() {
		cells := candidateCells(s, u, number)
		if len(cells) == 2 {
			pairs = append(pairs, conjugatePair{cells[0], cells[1], describeUnit(u)})
		}
	}
	return pairs
}

func (p conjugatePair) visualLink(number int) CandidateLink {
	return CandidateLink{From: p.a, To: p.b, Number: number, Unit: p.unit, Kind: "strong"}
}

// If at least one witness must contain number, every other cell seeing ALL
// witnesses loses that candidate. The witnesses themselves are left intact.
func commonCandidatePeers(s *Sudoku, witnesses []Cell, number int) []Cell {
	var targets []Cell
	for row := range 9 {
		for col := range 9 {
			cell := Cell{row, col}
			if !cellHasCandidate(s, cell, number) {
				continue
			}
			seesAll := len(witnesses) > 0
			for _, witness := range witnesses {
				seesAll = seesAll && cellsSeeEachOther(cell, witness)
			}
			if seesAll {
				targets = append(targets, cell)
			}
		}
	}
	return targets
}

func candidateConnection(from, to Cell, number int, kind string) CandidateLink {
	return CandidateLink{From: from, To: to, Number: number, Unit: sharedUnit(from, to), Kind: kind}
}

func linkEliminations(visual *StepVisual, targets, witnesses []Cell, number int) {
	for _, target := range targets {
		for _, witness := range witnesses {
			if cellsSeeEachOther(witness, target) {
				visual.CandidateLinks = append(visual.CandidateLinks, candidateConnection(witness, target, number, "elimination"))
			}
		}
	}
}

func applyPatternEliminations(s *Sudoku, technique, explanation string, number int, targets []Cell, details []string, visual *StepVisual) (Step, bool) {
	eliminations := removeCandidates(s, targets, number)
	if len(eliminations) == 0 {
		return Step{}, false
	}
	details = append(details, fmt.Sprintf("Eliminate %d from %s.", number, formatList(cellNames(targets))))
	return Step{
		Kind: "elimination", Technique: technique, Explanation: explanation,
		Details: details, Visual: visual, Eliminations: eliminations,
	}, true
}
