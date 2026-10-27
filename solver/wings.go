package solver

import "fmt"

func bivalueCells(s *Sudoku) []Cell {
	var cells []Cell
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] == 0 && candidateCount(s.Candidates[row][col]) == 2 {
				cells = append(cells, Cell{row, col})
			}
		}
	}
	return cells
}

// ApplyXYWing finds {X,Y} seeing both {X,Z} and {Y,Z}. One wing must be Z.
func ApplyXYWing(s *Sudoku) (Step, bool) { return applyPivotWing(s, false) }

// ApplyXYZWing uses a {X,Y,Z} pivot. A target must also see the pivot,
// because Z may be in the pivot rather than either wing.
func ApplyXYZWing(s *Sudoku) (Step, bool) { return applyPivotWing(s, true) }

func applyPivotWing(s *Sudoku, xyz bool) (Step, bool) {
	pivotSize, technique := 2, "XY-Wing"
	if xyz {
		pivotSize, technique = 3, "XYZ-Wing"
	}
	bivalues := bivalueCells(s)
	for row := range 9 {
		for col := range 9 {
			pivot := Cell{row, col}
			pivotMask := s.Candidates[row][col]
			if s.Board[row][col] != 0 || candidateCount(pivotMask) != pivotSize {
				continue
			}
			var wings []Cell
			for _, cell := range bivalues {
				mask := s.Candidates[cell.Row][cell.Col]
				if !cellsSeeEachOther(pivot, cell) {
					continue
				}
				if !xyz && candidateCount(mask&pivotMask) == 1 || xyz && mask&^pivotMask == 0 {
					wings = append(wings, cell)
				}
			}
			for i, a := range wings {
				for _, b := range wings[i+1:] {
					aMask, bMask := s.Candidates[a.Row][a.Col], s.Candidates[b.Row][b.Col]
					common := aMask & bMask
					if candidateCount(common) != 1 {
						continue
					}
					// The wings must cover the two different pivot choices. In an
					// XY-Wing their common digit is outside the pivot; in XYZ it is in it.
					if !xyz && (common&pivotMask != 0 || aMask|bMask != pivotMask|common) ||
						xyz && aMask|bMask != pivotMask {
						continue
					}
					number := singleCandidate(common)
					witnesses := []Cell{a, b}
					if xyz {
						witnesses = append(witnesses, pivot)
					}
					targets := commonCandidatePeers(s, witnesses, number)
					if len(targets) == 0 {
						continue
					}
					visual := &StepVisual{Sources: []Cell{pivot, a, b}, Digits: candidateList(pivotMask | common)}
					for _, wing := range []Cell{a, b} {
						for _, digit := range candidateList(pivotMask & s.Candidates[wing.Row][wing.Col]) {
							visual.CandidateLinks = append(visual.CandidateLinks, candidateConnection(pivot, wing, digit, "weak"))
						}
					}
					details := []string{fmt.Sprintf("Pivot %s allows only %v; wings %s and %s allow %v and %v.", cellName(row, col), candidateList(pivotMask), cellName(a.Row, a.Col), cellName(b.Row, b.Col), candidateList(aMask), candidateList(bMask))}
					for _, digit := range candidateList(pivotMask) {
						if digit == number {
							details = append(details, fmt.Sprintf("If the pivot is %d, it supplies %d itself.", digit, number))
							continue
						}
						wing := a
						if !hasCandidate(aMask, digit) {
							wing = b
						}
						details = append(details, fmt.Sprintf("If the pivot is %d, %s cannot be %d, so it must be %d.", digit, cellName(wing.Row, wing.Col), digit, number))
					}
					details = append(details, fmt.Sprintf("At least one of %s must be %d. A cell seeing all of them cannot be %d.", formatList(cellNames(witnesses)), number, number))
					linkEliminations(visual, targets, witnesses, number)
					return applyPatternEliminations(s, technique,
						fmt.Sprintf("%s at %s: at least one of %s must contain %d.", technique, cellName(row, col), formatList(cellNames(witnesses)), number),
						number, targets, details, visual)
				}
			}
		}
	}
	return Step{}, false
}

// ApplyWWing joins two identical bivalue cells by a conjugate pair on one
// of their digits. At least one bivalue cell must contain the other digit.
func ApplyWWing(s *Sudoku) (Step, bool) {
	bivalues := bivalueCells(s)
	for i, a := range bivalues {
		mask := s.Candidates[a.Row][a.Col]
		for _, b := range bivalues[i+1:] {
			if s.Candidates[b.Row][b.Col] != mask || cellsSeeEachOther(a, b) {
				continue // A shared-unit pair is handled by Naked Pair first.
			}
			for _, linkedDigit := range candidateList(mask) {
				number := singleCandidate(removeCandidate(mask, linkedDigit))
				targets := commonCandidatePeers(s, []Cell{a, b}, number)
				if len(targets) == 0 {
					continue
				}
				for _, pair := range conjugatePairs(s, linkedDigit) {
					if pair.a == a || pair.a == b || pair.b == a || pair.b == b {
						continue
					}
					for orientation := range 2 {
						p, q := pair.a, pair.b
						if orientation == 1 {
							p, q = q, p
						}
						if !cellsSeeEachOther(a, p) || !cellsSeeEachOther(b, q) {
							continue
						}
						visual := &StepVisual{
							Sources: []Cell{a, b, p, q}, Digits: candidateList(mask), Units: []VisualUnit{pair.unit},
							CandidateLinks: []CandidateLink{pair.visualLink(linkedDigit), candidateConnection(a, p, linkedDigit, "weak"), candidateConnection(b, q, linkedDigit, "weak")},
						}
						linkEliminations(visual, targets, []Cell{a, b}, number)
						details := []string{
							fmt.Sprintf("%s and %s both allow only %v.", cellName(a.Row, a.Col), cellName(b.Row, b.Col), candidateList(mask)),
							fmt.Sprintf("%s can place %d only in %s or %s.", unitLabel(pair.unit), linkedDigit, cellName(p.Row, p.Col), cellName(q.Row, q.Col)),
							fmt.Sprintf("If %s is %d, %s cannot be %d and must be %d.", cellName(p.Row, p.Col), linkedDigit, cellName(a.Row, a.Col), linkedDigit, number),
							fmt.Sprintf("If %s is %d, %s cannot be %d and must be %d.", cellName(q.Row, q.Col), linkedDigit, cellName(b.Row, b.Col), linkedDigit, number),
							fmt.Sprintf("Either way, at least one of those two bivalue cells is %d; remove %d from cells seeing both.", number, number),
						}
						return applyPatternEliminations(s, "W-Wing", fmt.Sprintf("W-Wing on %v: %s or %s must be %d.", candidateList(mask), cellName(a.Row, a.Col), cellName(b.Row, b.Col), number), number, targets, details, visual)
					}
				}
			}
		}
	}
	return Step{}, false
}
