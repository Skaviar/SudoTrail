package solver

import "fmt"

func rowNumbers(s *Sudoku, row int) []int {
	var numbers []int

	for col := range 9 {
		value := s.Board[row][col]

		if value != 0 {
			numbers = append(numbers, int(value))
		}
	}

	return numbers
}

func columnNumbers(s *Sudoku, col int) []int {
	var numbers []int

	for row := range 9 {
		value := s.Board[row][col]

		if value != 0 {
			numbers = append(numbers, int(value))
		}
	}

	return numbers
}

func boxNumbers(s *Sudoku, row, col int) []int {
	var numbers []int

	boxRow := (row / 3) * 3
	boxCol := (col / 3) * 3

	for r := boxRow; r < boxRow+3; r++ {
		for c := boxCol; c < boxCol+3; c++ {
			value := s.Board[r][c]

			if value != 0 {
				numbers = append(numbers, int(value))
			}
		}
	}

	return numbers
}

func candidatesAfterRemoving(
	candidates uint16,
	numbers []int,
) uint16 {
	for _, number := range numbers {
		candidates = removeCandidate(candidates, number)
	}

	return candidates
}

func explainCellCandidates(s *Sudoku, row, col int) []string {
	var details []string

	start := AllCandidates

	details = append(
		details,
		fmt.Sprintf(
			"R%dC%d starts with candidates: %v.",
			row+1,
			col+1,
			candidateList(start),
		),
	)

	rowNums := rowNumbers(s, row)
	afterRow := candidatesAfterRemoving(start, rowNums)

	details = append(
		details,
		fmt.Sprintf(
			"Row %d already contains %v → remove those numbers.",
			row+1,
			rowNums,
		),
	)

	details = append(
		details,
		fmt.Sprintf(
			"Candidates after checking Row %d: %v.",
			row+1,
			candidateList(afterRow),
		),
	)

	colNums := columnNumbers(s, col)
	afterColumn := candidatesAfterRemoving(afterRow, colNums)

	details = append(
		details,
		fmt.Sprintf(
			"Column %d already contains %v → remove those numbers.",
			col+1,
			colNums,
		),
	)

	details = append(
		details,
		fmt.Sprintf(
			"Candidates after checking Column %d: %v.",
			col+1,
			candidateList(afterColumn),
		),
	)

	boxNums := boxNumbers(s, row, col)
	afterBox := candidatesAfterRemoving(afterColumn, boxNums)

	boxNumber := (row/3)*3 + (col / 3) + 1

	details = append(
		details,
		fmt.Sprintf(
			"Box %d already contains %v → remove those numbers.",
			boxNumber,
			boxNums,
		),
	)

	details = append(
		details,
		fmt.Sprintf(
			"Final candidates for R%dC%d: %v.",
			row+1,
			col+1,
			candidateList(afterBox),
		),
	)

	return details
}

// --------------------------------------------------
// Hidden Single reasoning
// --------------------------------------------------

func explainHiddenSingleRow(
	s *Sudoku,
	row int,
	number int,
) []string {
	var details []string

	details = append(
		details,
		fmt.Sprintf(
			"We are looking for %d in Row %d.",
			number,
			row+1,
		),
	)

	for col := range 9 {
		if s.Board[row][col] != 0 {
			continue
		}

		if hasCandidate(s.Candidates[row][col], number) {
			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d → %d is possible because %d is not in Column %d and not in Box %d.",
					row+1,
					col+1,
					number,
					number,
					col+1,
					boxNumber(row, col),
				),
			)

			continue
		}

		reason := hiddenSingleEliminationReason(
			s,
			row,
			col,
			number,
		)

		details = append(
			details,
			fmt.Sprintf(
				"R%dC%d → %d is eliminated because %s.",
				row+1,
				col+1,
				number,
				reason,
			),
		)
	}

	return details
}

func explainHiddenSingleColumn(
	s *Sudoku,
	col int,
	number int,
) []string {
	var details []string

	details = append(
		details,
		fmt.Sprintf(
			"We are looking for %d in Column %d.",
			number,
			col+1,
		),
	)

	for row := range 9 {
		if s.Board[row][col] != 0 {
			continue
		}

		if hasCandidate(s.Candidates[row][col], number) {
			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d → %d is possible because %d is not in Row %d and not in Box %d.",
					row+1,
					col+1,
					number,
					number,
					row+1,
					boxNumber(row, col),
				),
			)

			continue
		}

		reason := hiddenSingleEliminationReason(
			s,
			row,
			col,
			number,
		)

		details = append(
			details,
			fmt.Sprintf(
				"R%dC%d → %d is eliminated because %s.",
				row+1,
				col+1,
				number,
				reason,
			),
		)
	}

	return details
}

func explainHiddenSingleBox(
	s *Sudoku,
	boxRow int,
	boxCol int,
	number int,
) []string {
	var details []string

	box := (boxRow/3)*3 + (boxCol / 3) + 1

	details = append(
		details,
		fmt.Sprintf(
			"We are looking for %d in Box %d.",
			number,
			box,
		),
	)

	for row := boxRow; row < boxRow+3; row++ {
		for col := boxCol; col < boxCol+3; col++ {
			if s.Board[row][col] != 0 {
				continue
			}

			if hasCandidate(s.Candidates[row][col], number) {
				details = append(
					details,
					fmt.Sprintf(
						"R%dC%d → %d is possible because %d is not in Row %d and not in Column %d.",
						row+1,
						col+1,
						number,
						number,
						row+1,
						col+1,
					),
				)

				continue
			}

			reason := hiddenSingleEliminationReason(
				s,
				row,
				col,
				number,
			)

			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d → %d is eliminated because %s.",
					row+1,
					col+1,
					number,
					reason,
				),
			)
		}
	}

	return details
}

// --------------------------------------------------
// Hidden Single elimination reasoning
// --------------------------------------------------

func hiddenSingleEliminationReason(
	s *Sudoku,
	row int,
	col int,
	number int,
) string {
	// Check row.
	for c := range 9 {
		if c == col {
			continue
		}

		if int(s.Board[row][c]) == number {
			return fmt.Sprintf(
				"%d is already in Row %d",
				number,
				row+1,
			)
		}
	}

	// Check column.
	for r := range 9 {
		if r == row {
			continue
		}

		if int(s.Board[r][col]) == number {
			return fmt.Sprintf(
				"%d is already in Column %d",
				number,
				col+1,
			)
		}
	}

	// Check box.
	boxRow := (row / 3) * 3
	boxCol := (col / 3) * 3

	for r := boxRow; r < boxRow+3; r++ {
		for c := boxCol; c < boxCol+3; c++ {
			if r == row && c == col {
				continue
			}

			if int(s.Board[r][c]) == number {
				return fmt.Sprintf(
					"%d is already in Box %d",
					number,
					boxNumber(row, col),
				)
			}
		}
	}

	return "the cell's row, column, or box prevents this placement"
}

// --------------------------------------------------
// Box helper
// --------------------------------------------------

func boxNumber(row, col int) int {
	return (row/3)*3 + (col / 3) + 1
}
