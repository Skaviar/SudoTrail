package solver

import (
	"fmt"
)

func FindFullHouse(s *Sudoku) (Step, bool) {
	// Check rows.
	for row := range 9 {
		emptyCol := -1

		for col := range 9 {
			if s.Board[row][col] == 0 {
				if emptyCol != -1 {
					// More than one empty cell.
					emptyCol = -2
					break
				}

				emptyCol = col
			}
		}

		if emptyCol >= 0 {
			number := singleMissingNumberRow(*s, row)

			if number == 0 {
				continue
			}

			s.Place(row, emptyCol, number)

			return Step{
				Technique: "Full House",
				Explanation: fmt.Sprintf(
					"Row %d has only one empty cell, R%dC%d, so the missing number is %d.",
					row+1,
					row+1,
					emptyCol+1,
					number,
				),
				Row:    row,
				Col:    emptyCol,
				Number: number,
			}, true
		}
	}

	// Check columns.
	for col := range 9 {
		emptyRow := -1

		for row := range 9 {
			if s.Board[row][col] == 0 {
				if emptyRow != -1 {
					// More than one empty cell.
					emptyRow = -2
					break
				}

				emptyRow = row
			}
		}

		if emptyRow >= 0 {
			number := singleMissingNumberColumn(s, col)

			if number == 0 {
				continue
			}

			s.Place(emptyRow, col, number)

			return Step{
				Technique: "Full House",
				Explanation: fmt.Sprintf(
					"Column %d has only one empty cell, R%dC%d, so the missing number is %d.",
					col+1,
					emptyRow+1,
					col+1,
					number,
				),
				Row:    emptyRow,
				Col:    col,
				Number: number,
			}, true
		}
	}

	// Check boxes.
	for boxRow := 0; boxRow < 9; boxRow += 3 {
		for boxCol := 0; boxCol < 9; boxCol += 3 {
			emptyRow := -1
			emptyCol := -1

			for row := boxRow; row < boxRow+3; row++ {
				for col := boxCol; col < boxCol+3; col++ {
					if s.Board[row][col] == 0 {
						if emptyRow != -1 {
							// More than one empty cell.
							emptyRow = -2
							break
						}

						emptyRow = row
						emptyCol = col
					}
				}

				if emptyRow == -2 {
					break
				}
			}

			if emptyRow >= 0 {
				number := singleMissingNumberBox(
					s,
					boxRow,
					boxCol,
				)

				if number == 0 {
					continue
				}

				boxNumber := (boxRow/3)*3 + (boxCol / 3) + 1

				s.Place(emptyRow, emptyCol, number)

				return Step{
					Technique: "Full House",
					Explanation: fmt.Sprintf(
						"Box %d has only one empty cell, R%dC%d, so the missing number is %d.",
						boxNumber,
						emptyRow+1,
						emptyCol+1,
						number,
					),
					Row:    emptyRow,
					Col:    emptyCol,
					Number: number,
				}, true
			}
		}
	}

	return Step{}, false
}

func singleMissingNumberRow(s Sudoku, row int) int {
	var seen [10]bool

	for col := range 9 {
		value := s.Board[row][col]

		if value != 0 {
			seen[value] = true
		}
	}

	for number := 1; number <= 9; number++ {
		if !seen[number] {
			return number
		}
	}

	return 0
}

func singleMissingNumberColumn(s *Sudoku, col int) int {
	var seen [10]bool

	for row := range 9 {
		value := s.Board[row][col]

		if value != 0 {
			seen[value] = true
		}
	}

	for number := 1; number <= 9; number++ {
		if !seen[number] {
			return number
		}
	}

	return 0
}

func singleMissingNumberBox(
	s *Sudoku,
	boxRow int,
	boxCol int,
) int {
	var seen [10]bool

	for row := boxRow; row < boxRow+3; row++ {
		for col := boxCol; col < boxCol+3; col++ {
			value := s.Board[row][col]

			if value != 0 {
				seen[value] = true
			}
		}
	}

	for number := 1; number <= 9; number++ {
		if !seen[number] {
			return number
		}
	}

	return 0
}
