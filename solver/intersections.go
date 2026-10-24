package solver

import "fmt"

// ApplyPointing finds a candidate confined to one row or column within a box.
// It removes that candidate from the rest of the row or column outside the box.
func ApplyPointing(s *Sudoku) (Step, bool) {
	for boxRow := 0; boxRow < 9; boxRow += 3 {
		for boxCol := 0; boxCol < 9; boxCol += 3 {
			box := boxNumber(boxRow, boxCol)

			for number := 1; number <= 9; number++ {
				var cells []Cell

				for row := boxRow; row < boxRow+3; row++ {
					for col := boxCol; col < boxCol+3; col++ {
						if s.Board[row][col] == 0 && hasCandidate(s.Candidates[row][col], number) {
							cells = append(cells, Cell{row, col})
						}
					}
				}

				// A single location is handled by Hidden Single first.
				if len(cells) < 2 {
					continue
				}

				sameRow := true
				sameCol := true
				for _, cell := range cells[1:] {
					if cell.Row != cells[0].Row {
						sameRow = false
					}
					if cell.Col != cells[0].Col {
						sameCol = false
					}
				}

				if sameRow {
					row := cells[0].Row
					var targets []Cell

					for col := 0; col < 9; col++ {
						if col >= boxCol && col < boxCol+3 {
							continue
						}
						if s.Board[row][col] == 0 && hasCandidate(s.Candidates[row][col], number) {
							targets = append(targets, Cell{row, col})
						}
					}

					if len(targets) > 0 {
						sourceNames := cellNames(cells)
						targetNames := cellNames(targets)
						details := []string{
							fmt.Sprintf("In Box %d, candidate %d appears in %s.", box, number, formatList(sourceNames)),
							fmt.Sprintf("All possible locations for %d in Box %d lie in Row %d.", number, box, row+1),
							fmt.Sprintf("Therefore, eliminate candidate %d from %s outside Box %d.", number, formatList(targetNames), box),
						}

						eliminations := removeCandidates(s, targets, number)
						return Step{
							Technique:    "Locked Candidates — Pointing",
							Explanation:  fmt.Sprintf("In Box %d, candidate %d is locked to Row %d.", box, number, row+1),
							Details:      details,
							Eliminations: eliminations,
						}, true
					}
				}

				if sameCol {
					col := cells[0].Col
					var targets []Cell

					for row := 0; row < 9; row++ {
						if row >= boxRow && row < boxRow+3 {
							continue
						}
						if s.Board[row][col] == 0 && hasCandidate(s.Candidates[row][col], number) {
							targets = append(targets, Cell{row, col})
						}
					}

					if len(targets) > 0 {
						sourceNames := cellNames(cells)
						targetNames := cellNames(targets)
						details := []string{
							fmt.Sprintf("In Box %d, candidate %d appears in %s.", box, number, formatList(sourceNames)),
							fmt.Sprintf("All possible locations for %d in Box %d lie in Column %d.", number, box, col+1),
							fmt.Sprintf("Therefore, eliminate candidate %d from %s outside Box %d.", number, formatList(targetNames), box),
						}

						eliminations := removeCandidates(s, targets, number)
						return Step{
							Technique:    "Locked Candidates — Pointing",
							Explanation:  fmt.Sprintf("In Box %d, candidate %d is locked to Column %d.", box, number, col+1),
							Details:      details,
							Eliminations: eliminations,
						}, true
					}
				}
			}
		}
	}

	return Step{}, false
}

// ApplyClaiming finds a candidate in a row or column whose possible locations
// are all inside one box. It removes that candidate from the rest of the box.
func ApplyClaiming(s *Sudoku) (Step, bool) {
	// Rows
	for row := 0; row < 9; row++ {
		for number := 1; number <= 9; number++ {
			var cells []Cell

			for col := 0; col < 9; col++ {
				if s.Board[row][col] == 0 && hasCandidate(s.Candidates[row][col], number) {
					cells = append(cells, Cell{row, col})
				}
			}

			if len(cells) < 2 {
				continue
			}

			boxCol := (cells[0].Col / 3) * 3
			sameBox := true
			for _, cell := range cells[1:] {
				if (cell.Col/3)*3 != boxCol {
					sameBox = false
					break
				}
			}
			if !sameBox {
				continue
			}

			boxRow := (row / 3) * 3
			box := boxNumber(boxRow, boxCol)
			var targets []Cell
			for r := boxRow; r < boxRow+3; r++ {
				if r == row {
					continue
				}
				for c := boxCol; c < boxCol+3; c++ {
					if s.Board[r][c] == 0 && hasCandidate(s.Candidates[r][c], number) {
						targets = append(targets, Cell{r, c})
					}
				}
			}

			if len(targets) == 0 {
				continue
			}

			details := []string{
				fmt.Sprintf("In Row %d, candidate %d appears in %s.", row+1, number, formatList(cellNames(cells))),
				fmt.Sprintf("All possible locations for %d in Row %d lie in Box %d.", number, row+1, box),
				fmt.Sprintf("Therefore, eliminate candidate %d from %s elsewhere in Box %d.", number, formatList(cellNames(targets)), box),
			}

			return Step{
				Technique:    "Locked Candidates — Claiming",
				Explanation:  fmt.Sprintf("In Row %d, candidate %d is confined to Box %d.", row+1, number, box),
				Details:      details,
				Eliminations: removeCandidates(s, targets, number),
			}, true
		}
	}

	// Columns
	for col := 0; col < 9; col++ {
		for number := 1; number <= 9; number++ {
			var cells []Cell

			for row := 0; row < 9; row++ {
				if s.Board[row][col] == 0 && hasCandidate(s.Candidates[row][col], number) {
					cells = append(cells, Cell{row, col})
				}
			}

			if len(cells) < 2 {
				continue
			}

			boxRow := (cells[0].Row / 3) * 3
			sameBox := true
			for _, cell := range cells[1:] {
				if (cell.Row/3)*3 != boxRow {
					sameBox = false
					break
				}
			}
			if !sameBox {
				continue
			}

			boxCol := (col / 3) * 3
			box := boxNumber(boxRow, boxCol)
			var targets []Cell
			for r := boxRow; r < boxRow+3; r++ {
				for c := boxCol; c < boxCol+3; c++ {
					if c == col {
						continue
					}
					if s.Board[r][c] == 0 && hasCandidate(s.Candidates[r][c], number) {
						targets = append(targets, Cell{r, c})
					}
				}
			}

			if len(targets) == 0 {
				continue
			}

			details := []string{
				fmt.Sprintf("In Column %d, candidate %d appears in %s.", col+1, number, formatList(cellNames(cells))),
				fmt.Sprintf("All possible locations for %d in Column %d lie in Box %d.", number, col+1, box),
				fmt.Sprintf("Therefore, eliminate candidate %d from %s elsewhere in Box %d.", number, formatList(cellNames(targets)), box),
			}

			return Step{
				Technique:    "Locked Candidates — Claiming",
				Explanation:  fmt.Sprintf("In Column %d, candidate %d is confined to Box %d.", col+1, number, box),
				Details:      details,
				Eliminations: removeCandidates(s, targets, number),
			}, true
		}
	}

	return Step{}, false
}

func removeCandidates(s *Sudoku, cells []Cell, number int) []CandidateElimination {
	eliminations := make([]CandidateElimination, 0, len(cells))
	for _, cell := range cells {
		row, col := cell.Row, cell.Col
		if s.Board[row][col] != 0 || !hasCandidate(s.Candidates[row][col], number) {
			continue
		}
		s.Candidates[row][col] = removeCandidate(s.Candidates[row][col], number)
		eliminations = append(eliminations, CandidateElimination{
			Row:    row,
			Col:    col,
			Number: number,
		})
	}
	return eliminations
}

func cellNames(cells []Cell) []string {
	names := make([]string, 0, len(cells))
	for _, cell := range cells {
		names = append(names, cellName(cell.Row, cell.Col))
	}
	return names
}
