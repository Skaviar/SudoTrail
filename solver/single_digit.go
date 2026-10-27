package solver

import "fmt"

// The three patterns join two conjugate pairs with a peer connection. If
// the first outside endpoint is false, the other outside endpoint is true.
func ApplySkyscraper(s *Sudoku) (Step, bool) { return applyTwoStringPattern(s, "Skyscraper") }
func ApplyTwoStringKite(s *Sudoku) (Step, bool) {
	return applyTwoStringPattern(s, "2-String Kite")
}
func ApplyTurbotFish(s *Sudoku) (Step, bool) { return applyTwoStringPattern(s, "Turbot Fish") }

func applyTwoStringPattern(s *Sudoku, technique string) (Step, bool) {
	for number := 1; number <= 9; number++ {
		pairs := conjugatePairs(s, number)
		for i, first := range pairs {
			for _, second := range pairs[i+1:] {
				if first.a == second.a || first.a == second.b || first.b == second.a || first.b == second.b {
					continue // A Turbot Fish has four distinct candidate cells.
				}
				if technique == "Skyscraper" && (first.unit.Kind == "box" || first.unit.Kind != second.unit.Kind) {
					continue
				}
				if technique == "2-String Kite" && !((first.unit.Kind == "row" && second.unit.Kind == "column") || (first.unit.Kind == "column" && second.unit.Kind == "row")) {
					continue
				}
				for orientation := range 4 {
					outerA, innerA := first.a, first.b
					innerB, outerB := second.a, second.b
					if orientation&1 != 0 {
						outerA, innerA = innerA, outerA
					}
					if orientation&2 != 0 {
						innerB, outerB = outerB, innerB
					}
					if !cellsSeeEachOther(innerA, innerB) {
						continue
					}
					if technique == "Skyscraper" {
						aligned := first.unit.Kind == "row" && innerA.Col == innerB.Col && outerA.Col != outerB.Col ||
							first.unit.Kind == "column" && innerA.Row == innerB.Row && outerA.Row != outerB.Row
						if !aligned {
							continue
						}
					}
					if technique == "2-String Kite" && boxNumber(innerA.Row, innerA.Col) != boxNumber(innerB.Row, innerB.Col) {
						continue
					}
					ends := []Cell{outerA, outerB}
					targets := commonCandidatePeers(s, ends, number)
					if len(targets) == 0 {
						continue
					}
					visual := &StepVisual{
						Sources: []Cell{outerA, innerA, innerB, outerB}, Digits: []int{number},
						Units:          []VisualUnit{first.unit, second.unit, sharedUnit(innerA, innerB)},
						CandidateLinks: []CandidateLink{first.visualLink(number), candidateConnection(innerA, innerB, number, "weak"), second.visualLink(number)},
					}
					linkEliminations(visual, targets, ends, number)
					details := []string{
						fmt.Sprintf("%s can place %d only in %s or %s.", unitLabel(first.unit), number, cellName(outerA.Row, outerA.Col), cellName(innerA.Row, innerA.Col)),
						fmt.Sprintf("%s can place %d only in %s or %s.", unitLabel(second.unit), number, cellName(innerB.Row, innerB.Col), cellName(outerB.Row, outerB.Col)),
						fmt.Sprintf("%s and %s share %s, so they cannot both contain %d.", cellName(innerA.Row, innerA.Col), cellName(innerB.Row, innerB.Col), unitLabel(sharedUnit(innerA, innerB)), number),
						fmt.Sprintf("If %s is not %d, %s must be %d; then %s cannot be %d, forcing %s to be %d.", cellName(outerA.Row, outerA.Col), number, cellName(innerA.Row, innerA.Col), number, cellName(innerB.Row, innerB.Col), number, cellName(outerB.Row, outerB.Col), number),
						fmt.Sprintf("At least one of %s is %d. Remove %d from cells seeing both endpoints.", formatList(cellNames(ends)), number, number),
					}
					return applyPatternEliminations(s, technique,
						fmt.Sprintf("%s on %d: %s or %s must contain %d.", technique, number, cellName(outerA.Row, outerA.Col), cellName(outerB.Row, outerB.Col), number),
						number, targets, details, visual)
				}
			}
		}
	}
	return Step{}, false
}

// ApplyEmptyRectangle finds a digit restricted to one row and one column in
// a box. An outside conjugate pair forces either its far endpoint or the
// remaining arm of the box to contain the digit.
func ApplyEmptyRectangle(s *Sudoku) (Step, bool) {
	units := allUnits()
	for number := 1; number <= 9; number++ {
		pairs := conjugatePairs(s, number)
		for box := range 9 {
			cells := candidateCells(s, units[18+box], number)
			// Two-candidate versions are already covered by Turbot Fish.
			if len(cells) < 3 || len(cells) > 5 {
				continue
			}
			for row := box / 3 * 3; row < box/3*3+3; row++ {
				for col := box % 3 * 3; col < box%3*3+3; col++ {
					var rowArm, columnArm []Cell
					covered := true
					for _, cell := range cells {
						if cell.Row != row && cell.Col != col {
							covered = false
						}
						if cell.Row == row && cell.Col != col {
							rowArm = append(rowArm, cell)
						}
						if cell.Col == col && cell.Row != row {
							columnArm = append(columnArm, cell)
						}
					}
					if !covered || len(rowArm) == 0 || len(columnArm) == 0 {
						continue
					}
					for _, pair := range pairs {
						if pair.unit.Kind == "box" {
							continue
						}
						for orientation := range 2 {
							near, far := pair.a, pair.b
							if orientation == 1 {
								near, far = far, near
							}
							if boxNumber(near.Row, near.Col) == box+1 {
								continue
							}
							var remaining []Cell
							var suppressed VisualUnit
							if pair.unit.Kind == "column" && near.Row == row {
								remaining, suppressed = columnArm, VisualUnit{Kind: "row", Index: row}
							} else if pair.unit.Kind == "row" && near.Col == col {
								remaining, suppressed = rowArm, VisualUnit{Kind: "column", Index: col}
							} else {
								continue
							}
							witnesses := append(append([]Cell(nil), remaining...), far)
							targets := commonCandidatePeers(s, witnesses, number)
							if len(targets) == 0 {
								continue
							}
							visual := &StepVisual{
								Sources: append(append([]Cell(nil), cells...), near, far), Digits: []int{number},
								Units:          []VisualUnit{{Kind: "box", Index: box}, pair.unit},
								CandidateLinks: []CandidateLink{pair.visualLink(number)},
							}
							for _, cell := range cells {
								if !containsCell(remaining, cell) {
									visual.CandidateLinks = append(visual.CandidateLinks, candidateConnection(near, cell, number, "weak"))
								}
							}
							linkEliminations(visual, targets, witnesses, number)
							details := []string{
								fmt.Sprintf("Box %d can place %d only along Row %d or Column %d: %s.", box+1, number, row+1, col+1, formatList(cellNames(cells))),
								fmt.Sprintf("%s can place %d only in %s or %s.", unitLabel(pair.unit), number, cellName(near.Row, near.Col), cellName(far.Row, far.Col)),
								fmt.Sprintf("If %s contains %d, cells seeing it cannot contain %d.", cellName(far.Row, far.Col), number, number),
								fmt.Sprintf("Otherwise %s must contain %d, removing %d from %s inside Box %d.", cellName(near.Row, near.Col), number, number, unitLabel(suppressed), box+1),
								fmt.Sprintf("Box %d must then place %d in %s. Targets seeing all these cells and %s lose %d in either case.", box+1, number, formatList(cellNames(remaining)), cellName(far.Row, far.Col), number),
							}
							return applyPatternEliminations(s, "Empty Rectangle", fmt.Sprintf("Empty Rectangle on %d in Box %d, linked through %s.", number, box+1, unitLabel(pair.unit)), number, targets, details, visual)
						}
					}
				}
			}
		}
	}
	return Step{}, false
}
