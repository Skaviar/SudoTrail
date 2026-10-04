package solver

import (
	"fmt"
)

func FindHiddenSingle(s *Sudoku) (Step, bool) {
	// Check rows.
	for row := range 9 {
		for number := 1; number <= 9; number++ {
			count := 0
			foundCol := -1

			for col := range 9 {
				if s.Board[row][col] != 0 {
					continue
				}

				if HasCandidate(
					s.Candidates[row][col],
					number,
				) {
					count++
					foundCol = col
				}
			}

			if count == 1 {
				s.Place(row, foundCol, number)

				return Step{
					Technique: "Hidden Single",
					Explanation: fmt.Sprintf(
						"In row %d, %d can only go in R%dC%d.",
						row+1,
						number,
						row+1,
						foundCol+1,
					),
					Row:    row,
					Col:    foundCol,
					Number: number,
				}, true
			}
		}
	}

	// Check columns.
	for col := range 9 {
		for number := 1; number <= 9; number++ {
			count := 0
			foundRow := -1

			for row := range 9 {
				if s.Board[row][col] != 0 {
					continue
				}

				if HasCandidate(
					s.Candidates[row][col],
					number,
				) {
					count++
					foundRow = row
				}
			}

			if count == 1 {
				s.Place(foundRow, col, number)

				return Step{
					Technique: "Hidden Single",
					Explanation: fmt.Sprintf(
						"In column %d, %d can only go in R%dC%d.",
						col+1,
						number,
						foundRow+1,
						col+1,
					),
					Row:    foundRow,
					Col:    col,
					Number: number,
				}, true
			}
		}
	}

	// Check boxes.
	for boxRow := 0; boxRow < 9; boxRow += 3 {
		for boxCol := 0; boxCol < 9; boxCol += 3 {
			boxNumber := (boxRow/3)*3 + (boxCol / 3) + 1

			for number := 1; number <= 9; number++ {
				count := 0
				foundRow := -1
				foundCol := -1

				for row := boxRow; row < boxRow+3; row++ {
					for col := boxCol; col < boxCol+3; col++ {
						if s.Board[row][col] != 0 {
							continue
						}

						if HasCandidate(
							s.Candidates[row][col],
							number,
						) {
							count++
							foundRow = row
							foundCol = col
						}
					}
				}

				if count == 1 {
					s.Place(foundRow, foundCol, number)

					return Step{
						Technique: "Hidden Single",
						Explanation: fmt.Sprintf(
							"In box %d, %d can only go in R%dC%d.",
							boxNumber,
							number,
							foundRow+1,
							foundCol+1,
						),
						Row:    foundRow,
						Col:    foundCol,
						Number: number,
					}, true
				}
			}
		}
	}

	return Step{}, false
}
