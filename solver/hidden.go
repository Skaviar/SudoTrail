package solver

import "fmt"

func (s *Sudoku) FindHiddenSingle() (Step, bool) {
	// Rows
	for row := range 9 {
		for number := 1; number <= 9; number++ {
			count := 0
			foundCol := -1

			for col := range 9 {
				if s.Board[row][col] != 0 {
					continue
				}

				if hasCandidate(s.Candidates[row][col], number) {
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

	// Columns
	for col := range 9 {
		for number := 1; number <= 9; number++ {
			count := 0
			foundRow := -1

			for row := range 9 {
				if s.Board[row][col] != 0 {
					continue
				}

				if hasCandidate(s.Candidates[row][col], number) {
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

	// Boxes
	for box := range 9 {
		boxRow := (box / 3) * 3
		boxCol := (box % 3) * 3

		for number := 1; number <= 9; number++ {
			count := 0
			foundRow := -1
			foundCol := -1

			for row := boxRow; row < boxRow+3; row++ {
				for col := boxCol; col < boxCol+3; col++ {
					if s.Board[row][col] != 0 {
						continue
					}

					if hasCandidate(s.Candidates[row][col], number) {
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
						box,
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

	return Step{}, false
}
