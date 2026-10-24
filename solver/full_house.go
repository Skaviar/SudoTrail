package solver

import "fmt"

// ApplyFullHouse fills the only empty cell in a unit.
func ApplyFullHouse(s *Sudoku) (Step, bool) {
	for _, unit := range allUnits() {
		var empty Cell
		emptyCount := 0
		missing := AllCandidates
		for _, cell := range unit.cells {
			value := s.Board[cell.Row][cell.Col]
			if value == 0 {
				empty = cell
				emptyCount++
			} else {
				missing = removeCandidate(missing, int(value))
			}
		}
		number := singleCandidate(missing)
		if emptyCount != 1 || number == 0 {
			continue
		}
		s.Place(empty.Row, empty.Col, number)
		return Step{
			Technique:   "Full House",
			Explanation: fmt.Sprintf("%s has only one empty cell, so its value is forced.", unit.name),
			Details: []string{
				fmt.Sprintf("Each digit from 1 through 9 must appear once in %s.", unit.name),
				fmt.Sprintf("The only missing digit is %d, so %s = %d.", number, cellName(empty.Row, empty.Col), number),
			},
			Row: empty.Row, Col: empty.Col, Number: number,
		}, true
	}
	return Step{}, false
}
