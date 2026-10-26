package solver

import "fmt"

// ApplyHiddenSingle places a digit with only one possible cell in a unit.
func ApplyHiddenSingle(s *Sudoku) (Step, bool) {
	for _, unit := range allUnits() {
		for number := 1; number <= 9; number++ {
			var target Cell
			count := 0
			for _, cell := range unit.cells {
				if s.Board[cell.Row][cell.Col] == 0 && hasCandidate(s.Candidates[cell.Row][cell.Col], number) {
					target = cell
					count++
				}
			}
			if count != 1 {
				continue
			}
			visual := hiddenSingleVisual(s, unit, target, number)
			details := explainHiddenSingle(s, unit, target, number)
			s.Place(target.Row, target.Col, number)
			return Step{
				Visual:      visual,
				Technique:   "Hidden Single",
				Explanation: fmt.Sprintf("In %s, number %d has only one possible location.", unit.name, number),
				Details:     details, Row: target.Row, Col: target.Col, Number: number,
			}, true
		}
	}
	return Step{}, false
}

func explainHiddenSingle(s *Sudoku, u unit, target Cell, number int) []string {
	visual := hiddenSingleVisual(s, u, target, number)
	details := []string{fmt.Sprintf("Find %d in %s.", number, u.name)}
	for _, restriction := range visual.Restrictions {
		cell := Cell{Row: restriction.Row, Col: restriction.Col}
		if link, ok := blockingNumber(s, cell, number); ok {
			details = append(details, fmt.Sprintf("%s already contains %d. Both cells share %s, so %s cannot contain %d.", cellName(link.From.Row, link.From.Col), number, unitLabel(link.Unit), cellName(cell.Row, cell.Col), number))
		} else {
			details = append(details, fmt.Sprintf("An earlier solving step ruled out %d in %s.", number, cellName(cell.Row, cell.Col)))
		}
	}
	return append(details, fmt.Sprintf("Only %s remains, so place %d there.", cellName(target.Row, target.Col), number))
}
