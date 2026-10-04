package solver

import "fmt"

func FindHiddenSingle(s *Sudoku) (Step, bool) {
	// --------------------------------------------------
	// Rows
	// --------------------------------------------------
	for row := range 9 {
		for number := 1; number <= 9; number++ {
			var possibleCells []int

			for col := range 9 {
				if s.Board[row][col] == 0 &&
					hasCandidate(s.Candidates[row][col], number) {
					possibleCells = append(possibleCells, col)
				}
			}

			if len(possibleCells) == 1 {
				col := possibleCells[0]

				details := explainHiddenSingleRow(
					s,
					row,
					number,
				)

				details = append(
					details,
					fmt.Sprintf(
						"Therefore R%dC%d = %d.",
						row+1,
						col+1,
						number,
					),
				)

				s.Place(row, col, number)

				return Step{
					Technique: "Hidden Single",
					Explanation: fmt.Sprintf(
						"In Row %d, number %d has only one possible location.",
						row+1,
						number,
					),
					Details: details,
					Row:     row,
					Col:     col,
					Number:  number,
				}, true
			}
		}
	}

	// --------------------------------------------------
	// Columns
	// --------------------------------------------------
	for col := range 9 {
		for number := 1; number <= 9; number++ {
			var possibleCells []int

			for row := range 9 {
				if s.Board[row][col] == 0 &&
					hasCandidate(s.Candidates[row][col], number) {
					possibleCells = append(possibleCells, row)
				}
			}

			if len(possibleCells) == 1 {
				row := possibleCells[0]

				details := explainHiddenSingleColumn(
					s,
					col,
					number,
				)

				details = append(
					details,
					fmt.Sprintf(
						"Therefore R%dC%d = %d.",
						row+1,
						col+1,
						number,
					),
				)

				s.Place(row, col, number)

				return Step{
					Technique: "Hidden Single",
					Explanation: fmt.Sprintf(
						"In Column %d, number %d has only one possible location.",
						col+1,
						number,
					),
					Details: details,
					Row:     row,
					Col:     col,
					Number:  number,
				}, true
			}
		}
	}

	// --------------------------------------------------
	// Boxes
	// --------------------------------------------------
	for boxRow := 0; boxRow < 9; boxRow += 3 {
		for boxCol := 0; boxCol < 9; boxCol += 3 {
			boxNumber := (boxRow/3)*3 + (boxCol / 3) + 1

			for number := 1; number <= 9; number++ {
				foundRow := -1
				foundCol := -1
				count := 0

				for row := boxRow; row < boxRow+3; row++ {
					for col := boxCol; col < boxCol+3; col++ {
						if s.Board[row][col] == 0 &&
							hasCandidate(s.Candidates[row][col], number) {
							count++
							foundRow = row
							foundCol = col
						}
					}
				}

				if count == 1 {
					details := explainHiddenSingleBox(
						s,
						boxRow,
						boxCol,
						number,
					)

					details = append(
						details,
						fmt.Sprintf(
							"Therefore R%dC%d = %d.",
							foundRow+1,
							foundCol+1,
							number,
						),
					)

					s.Place(foundRow, foundCol, number)

					return Step{
						Technique: "Hidden Single",
						Explanation: fmt.Sprintf(
							"In Box %d, number %d has only one possible location.",
							boxNumber,
							number,
						),
						Details: details,
						Row:     foundRow,
						Col:     foundCol,
						Number:  number,
					}, true
				}
			}
		}
	}

	return Step{}, false
}
