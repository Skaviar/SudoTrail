package solver

import (
	"fmt"
	"strings"
)

func FindHiddenSingle(s *Sudoku) (Step, bool) {
	// --------------------------------------------------
	// Rows
	// --------------------------------------------------
	for row := range 9 {
		for number := 1; number <= 9; number++ {
			var possibleCells []string
			foundCol := -1

			for col := range 9 {
				if s.Board[row][col] != 0 {
					continue
				}

				if hasCandidate(s.Candidates[row][col], number) {
					possibleCells = append(
						possibleCells,
						fmt.Sprintf("R%dC%d", row+1, col+1),
					)
					foundCol = col
				}
			}

			if len(possibleCells) == 1 {
				details := []string{
					fmt.Sprintf(
						"We are looking for number %d in Row %d.",
						number,
						row+1,
					),
					fmt.Sprintf(
						"Number %d can potentially go in: %s.",
						number,
						strings.Join(possibleCells, ", "),
					),
					fmt.Sprintf(
						"Every other empty cell in Row %d is unable to contain %d.",
						row+1,
						number,
					),
					fmt.Sprintf(
						"Therefore R%dC%d = %d.",
						row+1,
						foundCol+1,
						number,
					),
				}

				s.Place(row, foundCol, number)

				return Step{
					Technique: "Hidden Single",
					Explanation: fmt.Sprintf(
						"In Row %d, %d has only one possible location.",
						row+1,
						number,
					),
					Details: details,
					Row:     row,
					Col:     foundCol,
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
			var possibleCells []string
			foundRow := -1

			for row := range 9 {
				if s.Board[row][col] != 0 {
					continue
				}

				if hasCandidate(s.Candidates[row][col], number) {
					possibleCells = append(
						possibleCells,
						fmt.Sprintf("R%dC%d", row+1, col+1),
					)
					foundRow = row
				}
			}

			if len(possibleCells) == 1 {
				details := []string{
					fmt.Sprintf(
						"We are looking for number %d in Column %d.",
						number,
						col+1,
					),
					fmt.Sprintf(
						"Number %d can potentially go in: %s.",
						number,
						strings.Join(possibleCells, ", "),
					),
					fmt.Sprintf(
						"Every other empty cell in Column %d is unable to contain %d.",
						col+1,
						number,
					),
					fmt.Sprintf(
						"Therefore R%dC%d = %d.",
						foundRow+1,
						col+1,
						number,
					),
				}

				s.Place(foundRow, col, number)

				return Step{
					Technique: "Hidden Single",
					Explanation: fmt.Sprintf(
						"In Column %d, %d has only one possible location.",
						col+1,
						number,
					),
					Details: details,
					Row:     foundRow,
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
				var possibleCells []string
				foundRow := -1
				foundCol := -1

				for row := boxRow; row < boxRow+3; row++ {
					for col := boxCol; col < boxCol+3; col++ {
						if s.Board[row][col] != 0 {
							continue
						}

						if hasCandidate(s.Candidates[row][col], number) {
							possibleCells = append(
								possibleCells,
								fmt.Sprintf("R%dC%d", row+1, col+1),
							)
							foundRow = row
							foundCol = col
						}
					}
				}

				if len(possibleCells) == 1 {
					details := []string{
						fmt.Sprintf(
							"We are looking for number %d in Box %d.",
							number,
							boxNumber,
						),
						fmt.Sprintf(
							"Number %d can potentially go in: %s.",
							number,
							strings.Join(possibleCells, ", "),
						),
						fmt.Sprintf(
							"Every other empty cell in Box %d is unable to contain %d.",
							boxNumber,
							number,
						),
						fmt.Sprintf(
							"Therefore R%dC%d = %d.",
							foundRow+1,
							foundCol+1,
							number,
						),
					}

					s.Place(foundRow, foundCol, number)

					return Step{
						Technique: "Hidden Single",
						Explanation: fmt.Sprintf(
							"In Box %d, %d has only one possible location.",
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
