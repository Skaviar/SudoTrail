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
			details := explainHiddenSingle(s, unit, target, number)
			s.Place(target.Row, target.Col, number)
			return Step{
				Technique:   "Hidden Single",
				Explanation: fmt.Sprintf("In %s, number %d has only one possible location.", unit.name, number),
				Details:     details, Row: target.Row, Col: target.Col, Number: number,
			}, true
		}
	}
	return Step{}, false
}

func explainHiddenSingle(s *Sudoku, unit unit, target Cell, number int) []string {
	details := []string{fmt.Sprintf("We are looking for %d in %s.", number, unit.name)}
	// Group blocked cells by the first unit that excludes the digit.
	// The fixed unit order keeps explanations deterministic.
	groups := make(map[string][]string)
	var reasons []string
	for _, cell := range unit.cells {
		if s.Board[cell.Row][cell.Col] != 0 || cell == target {
			continue
		}
		reason := "Earlier solving steps removed this candidate"
		for _, peerUnit := range allUnits() {
			containsCell, containsNumber := false, false
			for _, peer := range peerUnit.cells {
				if peer == cell {
					containsCell = true
				}
				if int(s.Board[peer.Row][peer.Col]) == number {
					containsNumber = true
				}
			}
			if containsCell && containsNumber {
				reason = fmt.Sprintf("%d is already in %s", number, peerUnit.name)
				break
			}
		}
		if _, exists := groups[reason]; !exists {
			reasons = append(reasons, reason)
		}
		groups[reason] = append(groups[reason], cellName(cell.Row, cell.Col))
	}
	for _, reason := range reasons {
		details = append(details, fmt.Sprintf("%s → eliminate %s.", reason, formatList(groups[reason])))
	}
	return append(details, fmt.Sprintf("Only %s remains, so %s = %d.", cellName(target.Row, target.Col), cellName(target.Row, target.Col), number))
}
