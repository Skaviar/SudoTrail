package solver

import "fmt"

func FindFullHouse(s *Sudoku) (Step, bool) {
	// --------------------------------------------------
	// Rows
	// --------------------------------------------------
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
			number := singleMissingNumberRow(s, row)

			if number == 0 {
				continue
			}

			present := presentNumbersRow(s, row)

			details := []string{
				fmt.Sprintf(
					"Row %d currently contains: %v.",
					row+1,
					present,
				),
				fmt.Sprintf(
					"R%dC%d is the only empty cell in Row %d.",
					row+1,
					emptyCol+1,
					row+1,
				),
				fmt.Sprintf(
					"A Sudoku row must contain every number from 1 through 9 exactly once.",
				),
				fmt.Sprintf(
					"The only number missing from Row %d is %d.",
					row+1,
					number,
				),
				fmt.Sprintf(
					"Therefore R%dC%d = %d.",
					row+1,
					emptyCol+1,
					number,
				),
			}

			s.Place(row, emptyCol, number)

			return Step{
				Technique: "Full House",
				Explanation: fmt.Sprintf(
					"Row %d has only one empty cell, so its value is forced.",
					row+1,
				),
				Details: details,
				Row:     row,
				Col:     emptyCol,
				Number:  number,
			}, true
		}
	}

	// --------------------------------------------------
	// Columns
	// --------------------------------------------------
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

			present := presentNumbersColumn(s, col)

			details := []string{
				fmt.Sprintf(
					"Column %d currently contains: %v.",
					col+1,
					present,
				),
				fmt.Sprintf(
					"R%dC%d is the only empty cell in Column %d.",
					emptyRow+1,
					col+1,
					col+1,
				),
				fmt.Sprintf(
					"A Sudoku column must contain every number from 1 through 9 exactly once.",
				),
				fmt.Sprintf(
					"The only number missing from Column %d is %d.",
					col+1,
					number,
				),
				fmt.Sprintf(
					"Therefore R%dC%d = %d.",
					emptyRow+1,
					col+1,
					number,
				),
			}

			s.Place(emptyRow, col, number)

			return Step{
				Technique: "Full House",
				Explanation: fmt.Sprintf(
					"Column %d has only one empty cell, so its value is forced.",
					col+1,
				),
				Details: details,
				Row:     emptyRow,
				Col:     col,
				Number:  number,
			}, true
		}
	}

	// --------------------------------------------------
	// Boxes
	// --------------------------------------------------
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
				present := presentNumbersBox(s, boxRow, boxCol)

				details := []string{
					fmt.Sprintf(
						"Box %d currently contains: %v.",
						boxNumber,
						present,
					),
					fmt.Sprintf(
						"R%dC%d is the only empty cell in Box %d.",
						emptyRow+1,
						emptyCol+1,
						boxNumber,
					),
					"A Sudoku box must contain every number from 1 through 9 exactly once.",
					fmt.Sprintf(
						"The only number missing from Box %d is %d.",
						boxNumber,
						number,
					),
					fmt.Sprintf(
						"Therefore R%dC%d = %d.",
						emptyRow+1,
						emptyCol+1,
						number,
					),
				}

				s.Place(emptyRow, emptyCol, number)

				return Step{
					Technique: "Full House",
					Explanation: fmt.Sprintf(
						"Box %d has only one empty cell, so its value is forced.",
						boxNumber,
					),
					Details: details,
					Row:     emptyRow,
					Col:     emptyCol,
					Number:  number,
				}, true
			}
		}
	}

	return Step{}, false
}

// --------------------------------------------------
// Missing number helpers
// --------------------------------------------------

func singleMissingNumberRow(s *Sudoku, row int) int {
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

// --------------------------------------------------
// Present number helpers
// --------------------------------------------------

func presentNumbersRow(s *Sudoku, row int) []int {
	var numbers []int

	for number := 1; number <= 9; number++ {
		for col := range 9 {
			if int(s.Board[row][col]) == number {
				numbers = append(numbers, number)
				break
			}
		}
	}

	return numbers
}

func presentNumbersColumn(s *Sudoku, col int) []int {
	var numbers []int

	for number := 1; number <= 9; number++ {
		for row := range 9 {
			if int(s.Board[row][col]) == number {
				numbers = append(numbers, number)
				break
			}
		}
	}

	return numbers
}

func presentNumbersBox(
	s *Sudoku,
	boxRow int,
	boxCol int,
) []int {
	var numbers []int

	for number := 1; number <= 9; number++ {
		found := false

		for row := boxRow; row < boxRow+3; row++ {
			for col := boxCol; col < boxCol+3; col++ {
				if int(s.Board[row][col]) == number {
					found = true
					break
				}
			}

			if found {
				break
			}
		}

		if found {
			numbers = append(numbers, number)
		}
	}

	return numbers
}
