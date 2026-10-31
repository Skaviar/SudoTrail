package solver

import "fmt"

// ApplyUniqueRectangle supports types 1..6. Types 1/2 have small direct scans;
// the other types share the rectangle scanner. Missing variants are separate.
func ApplyUniqueRectangle(s *Sudoku, kind int) (Step, bool) {
	if kind >= 3 && kind <= 6 {
		return applyRectangles(s, false, fmt.Sprintf("Unique Rectangle Type %d", kind))
	}
	if kind != 1 && kind != 2 {
		return Step{}, false
	}
	for top := 0; top < 8; top++ {
		for bottom := top + 1; bottom < 9; bottom++ {
			for left := 0; left < 8; left++ {
				for right := left + 1; right < 9; right++ {
					// Exactly two boxes: either the rows share a band, or the
					// columns share a stack, but not both.
					if (top/3 == bottom/3) == (left/3 == right/3) {
						continue
					}
					cells := []Cell{{top, left}, {top, right}, {bottom, left}, {bottom, right}}
					pair, roofs, ok := rectangleCandidates(s, cells)
					if !ok || len(roofs) != kind {
						continue
					}
					technique := fmt.Sprintf("Unique Rectangle Type %d", kind)
					visual := rectangleVisual(cells, pair)
					details := []string{
						fmt.Sprintf("%s form a rectangle in exactly two rows, two columns, and two boxes on digits %v.", formatList(cellNames(cells)), candidateList(pair)),
						"The original submitted clues have been verified to have exactly one solution.",
						"If all four cells used only the rectangle digits, those digits could be swapped to create another solution.",
					}
					if kind == 1 {
						if !s.HasUniqueSolution() {
							return Step{}, false
						}
						roof := roofs[0]
						var eliminations []CandidateElimination
						for _, number := range candidateList(pair) {
							eliminations = append(eliminations, removeCandidates(s, []Cell{roof}, number)...)
							details = append(details, fmt.Sprintf("Eliminate %d from %s.", number, cellName(roof.Row, roof.Col)))
						}
						return Step{Kind: "elimination", Technique: technique,
							Explanation: fmt.Sprintf("%s must use an extra candidate to avoid a second solution.", cellName(roof.Row, roof.Col)),
							Details:     details, Eliminations: eliminations, Visual: visual}, true
					}
					// Type 2 has adjacent roofs, each with the same one extra digit.
					if roofs[0].Row != roofs[1].Row && roofs[0].Col != roofs[1].Col {
						continue
					}
					extra := s.Candidates[roofs[0].Row][roofs[0].Col] &^ pair
					if candidateCount(extra) != 1 || s.Candidates[roofs[1].Row][roofs[1].Col]&^pair != extra {
						continue
					}
					number := singleCandidate(extra)
					targets := commonCandidatePeers(s, roofs, number)
					if len(targets) == 0 {
						continue
					}
					if !s.HasUniqueSolution() {
						return Step{}, false
					}
					visual.Digits = candidateList(pair | extra)
					linkEliminations(visual, targets, roofs, number)
					details = append(details, fmt.Sprintf("At least one of %s must be %d; remove %d from other cells seeing both.", formatList(cellNames(roofs)), number, number))
					return applyPatternEliminations(s, technique, fmt.Sprintf("One of %s must contain %d to avoid a second solution.", formatList(cellNames(roofs)), number), number, targets, details, visual)
				}
			}
		}
	}
	return Step{}, false
}

func rectangleCandidates(s *Sudoku, cells []Cell) (uint16, []Cell, bool) {
	var pair uint16
	for _, cell := range cells {
		if s.Board[cell.Row][cell.Col] != 0 {
			return 0, nil, false
		}
		if mask := s.Candidates[cell.Row][cell.Col]; candidateCount(mask) == 2 {
			pair = mask
		}
	}
	if pair == 0 {
		return 0, nil, false
	}
	var roofs []Cell
	for _, cell := range cells {
		mask := s.Candidates[cell.Row][cell.Col]
		if mask&pair != pair {
			return 0, nil, false
		}
		if mask != pair {
			roofs = append(roofs, cell)
		}
	}
	return pair, roofs, true
}

func rectangleVisual(cells []Cell, pair uint16) *StepVisual {
	v := &StepVisual{Sources: cells, Digits: candidateList(pair)}
	for _, edge := range [][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
		for _, number := range candidateList(pair) {
			v.CandidateLinks = append(v.CandidateLinks, candidateConnection(cells[edge[0]], cells[edge[1]], number, "weak"))
		}
	}
	return v
}

// ApplyBUGPlusOne checks the full BUG condition after removing one candidate
// from the only trivalue cell. Merely finding one trivalue cell is insufficient.
func ApplyBUGPlusOne(s *Sudoku) (Step, bool) {
	var pivot Cell
	found := false
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] != 0 {
				continue
			}
			count := candidateCount(s.Candidates[row][col])
			if count == 3 && !found {
				pivot, found = Cell{row, col}, true
			} else if count != 2 {
				return Step{}, false
			}
		}
	}
	if !found {
		return Step{}, false
	}
	for _, number := range candidateList(s.Candidates[pivot.Row][pivot.Col]) {
		if !wouldBeBUG(s, pivot, number) {
			continue
		}
		if !s.HasUniqueSolution() {
			return Step{}, false
		}
		before := s.Candidates[pivot.Row][pivot.Col]
		v := &StepVisual{Sources: []Cell{pivot}, Digits: []int{number}}
		for _, u := range allUnits() {
			if !containsCell(u.cells, pivot) {
				continue
			}
			v.Units = append(v.Units, describeUnit(u))
			for _, cell := range candidateCells(s, u, number) {
				if !containsCell(v.Sources, cell) {
					v.Sources = append(v.Sources, cell)
				}
			}
		}
		s.Place(pivot.Row, pivot.Col, number)
		return Step{Kind: "placement", Technique: "BUG+1", Row: pivot.Row, Col: pivot.Col, Number: number, BeforeCandidates: before, Visual: v,
			Explanation: fmt.Sprintf("Place %d in %s to avoid a Binary Universal Grave.", number, cellName(pivot.Row, pivot.Col)),
			Details: []string{"The original submitted clues have been verified to have exactly one solution.",
				"Every other empty cell has exactly two candidates.",
				fmt.Sprintf("Removing %d from %s would leave every unplaced digit exactly twice in each row, column, and box: a BUG with no unique solution.", number, cellName(pivot.Row, pivot.Col)),
				fmt.Sprintf("Therefore %s must contain %d.", cellName(pivot.Row, pivot.Col), number)}}, true
	}
	return Step{}, false
}

func wouldBeBUG(s *Sudoku, pivot Cell, extra int) bool {
	for _, u := range allUnits() {
		var counts [10]int
		var placed uint16
		for _, cell := range u.cells {
			if number := s.Board[cell.Row][cell.Col]; number != 0 {
				placed |= 1 << number
				continue
			}
			mask := s.Candidates[cell.Row][cell.Col]
			if cell == pivot {
				mask = removeCandidate(mask, extra)
			}
			for _, number := range candidateList(mask) {
				counts[number]++
			}
		}
		for number := 1; number <= 9; number++ {
			want := 2
			if hasCandidate(placed, number) {
				want = 0
			}
			if counts[number] != want {
				return false
			}
		}
	}
	return true
}
