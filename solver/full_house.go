package solver

import "fmt"

func (s *Sudoku) FindFullHouse() (Step, bool) {
	// Rows
	for row := range 9 {
		emptyCol := -1
		emptyCount := 0

		for col := range 9 {
			if s.Board[row][col] == 0 {
				emptyCol = col
				emptyCount++
			}
		}

		if emptyCount == 1 {
			number := singleMissingNumberRow(s.Board[row])

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

	// Columns
	for col := range 9 {
		emptyRow := -1
		emptyCount := 0

		for row := range 9 {
			if s.Board[row][col] == 0 {
				emptyRow = row
				emptyCount++
			}
		}

		if emptyCount == 1 {
			number := s.singleMissingNumberColumn(col)

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

	// Boxes
	for box := range 9 {
		emptyRow := -1
		emptyCol := -1
		emptyCount := 0

		boxRow := (box / 3) * 3
		boxCol := (box % 3) * 3

		for row := boxRow; row < boxRow+3; row++ {
			for col := boxCol; col < boxCol+3; col++ {
				if s.Board[row][col] == 0 {
					emptyRow = row
					emptyCol = col
					emptyCount++
				}
			}
		}

		if emptyCount == 1 {
			number := s.singleMissingNumberBox(box)

			s.Place(emptyRow, emptyCol, number)

			return Step{
				Technique: "Full House",
				Explanation: fmt.Sprintf(
					"Box %d has only one empty cell, R%dC%d, so the missing number is %d.",
					box,
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

	return Step{}, false
}

func singleMissingNumberRow(row [9]int8) int {
	var seen [10]bool

	for _, value := range row {
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

func (s *Sudoku) singleMissingNumberColumn(col int) int {
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

func (s *Sudoku) singleMissingNumberBox(box int) int {
	var seen [10]bool

	boxRow := (box / 3) * 3
	boxCol := (box % 3) * 3

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
