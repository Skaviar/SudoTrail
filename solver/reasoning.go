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

	// Row elimination
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

	// Column elimination
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

	// Box elimination
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

func explainHiddenSingleRow(
	s *Sudoku,
	row int,
	number int,
) []string {
	var details []string

	details = append(
		details,
		fmt.Sprintf(
			"We are trying to place %d in Row %d.",
			number,
			row+1,
		),
	)

	for col := range 9 {
		if s.Board[row][col] != 0 {
			continue
		}

		candidates := s.Candidates[row][col]

		if hasCandidate(candidates, number) {
			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d has candidates %v → %d is possible.",
					row+1,
					col+1,
					candidateList(candidates),
					number,
				),
			)
		} else {
			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d has candidates %v → %d is impossible.",
					row+1,
					col+1,
					candidateList(candidates),
					number,
				),
			)
		}
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
			"We are trying to place %d in Column %d.",
			number,
			col+1,
		),
	)

	for row := range 9 {
		if s.Board[row][col] != 0 {
			continue
		}

		candidates := s.Candidates[row][col]

		if hasCandidate(candidates, number) {
			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d has candidates %v → %d is possible.",
					row+1,
					col+1,
					candidateList(candidates),
					number,
				),
			)
		} else {
			details = append(
				details,
				fmt.Sprintf(
					"R%dC%d has candidates %v → %d is impossible.",
					row+1,
					col+1,
					candidateList(candidates),
					number,
				),
			)
		}
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

	boxNumber := (boxRow/3)*3 + (boxCol / 3) + 1

	details = append(
		details,
		fmt.Sprintf(
			"We are trying to place %d in Box %d.",
			number,
			boxNumber,
		),
	)

	for row := boxRow; row < boxRow+3; row++ {
		for col := boxCol; col < boxCol+3; col++ {
			if s.Board[row][col] != 0 {
				continue
			}

			candidates := s.Candidates[row][col]

			if hasCandidate(candidates, number) {
				details = append(
					details,
					fmt.Sprintf(
						"R%dC%d has candidates %v → %d is possible.",
						row+1,
						col+1,
						candidateList(candidates),
						number,
					),
				)
			} else {
				details = append(
					details,
					fmt.Sprintf(
						"R%dC%d has candidates %v → %d is impossible.",
						row+1,
						col+1,
						candidateList(candidates),
						number,
					),
				)
			}
		}
	}

	return details
}
