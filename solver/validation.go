package solver

import "fmt"

// ValidationIssue identifies conflicting clues or an impossible empty cell.
// Coordinates are zero-based, matching Board and Candidates.
type ValidationIssue struct {
	Message string `json:"message"`
	Cells   []Cell `json:"cells"`
}

// ValidatePuzzle checks local Sudoku rules, not solvability or uniqueness.
func ValidatePuzzle(puzzle string) []ValidationIssue {
	issues := []ValidationIssue{}
	if len(puzzle) != 81 {
		return append(issues, ValidationIssue{Message: "puzzle must contain exactly 81 characters", Cells: []Cell{}})
	}
	s := &Sudoku{}
	for index, ch := range []byte(puzzle) {
		if ch == '.' || ch == '0' {
			continue
		}
		cell := Cell{Row: index / 9, Col: index % 9}
		if ch < '1' || ch > '9' {
			issues = append(issues, ValidationIssue{Message: "use only digits 1–9, 0, or .", Cells: []Cell{cell}})
			continue
		}
		s.Board[cell.Row][cell.Col] = int8(ch - '0')
	}
	if len(issues) > 0 {
		return issues
	}
	for _, unit := range allUnits() {
		for number := 1; number <= 9; number++ {
			cells := []Cell{}
			for _, cell := range unit.cells {
				if int(s.Board[cell.Row][cell.Col]) == number {
					cells = append(cells, cell)
				}
			}
			if len(cells) > 1 {
				issues = append(issues, ValidationIssue{Message: fmt.Sprintf("duplicate %d in %s", number, unit.name), Cells: cells})
			}
		}
	}
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] == 0 && s.calculateCandidates(row, col) == 0 {
				issues = append(issues, ValidationIssue{Message: fmt.Sprintf("%s has no possible digits", cellName(row, col)), Cells: []Cell{{Row: row, Col: col}}})
			}
		}
	}
	return issues
}
