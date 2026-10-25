package solver

import "fmt"

func ApplyNakedSingle(s *Sudoku) (Step, bool) {
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] != 0 {
				continue
			}

			candidates := s.Candidates[row][col]

			if candidateCount(candidates) != 1 {
				continue
			}

			number := singleCandidate(candidates)

			details := explainCellCandidates(s, row, col)

			details = append(
				details,
				fmt.Sprintf(
					"Only %d remains, so R%dC%d must be %d.",
					number,
					row+1,
					col+1,
					number,
				),
			)

			s.Place(row, col, number)

			return Step{
				Visual:           &StepVisual{Sources: []Cell{{Row: row, Col: col}}, Digits: []int{number}},
				Technique:        "Naked Single",
				Explanation:      fmt.Sprintf("R%dC%d has only one possible candidate.", row+1, col+1),
				Details:          details,
				Row:              row,
				Col:              col,
				Number:           number,
				BeforeCandidates: candidates,
			}, true
		}
	}

	return Step{}, false
}

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

// --------------------------------------------------
// Candidate helpers
// --------------------------------------------------

func candidatesAfterRemoving(
	candidates uint16,
	numbers []int,
) uint16 {
	for _, number := range numbers {
		candidates = removeCandidate(candidates, number)
	}

	return candidates
}

// --------------------------------------------------
// Naked Single reasoning
// --------------------------------------------------

func explainCellCandidates(
	s *Sudoku,
	row int,
	col int,
) []string {
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

	// Row
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

	// Column
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

	// Box
	boxNums := boxNumbers(s, row, col)
	afterBox := candidatesAfterRemoving(afterColumn, boxNums)

	details = append(
		details,
		fmt.Sprintf(
			"Box %d already contains %v → remove those numbers.",
			boxNumber(row, col),
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

	if afterBox != s.Candidates[row][col] {
		details = append(details, fmt.Sprintf("Earlier solving steps removed %v; remaining candidates: %v.", candidateList(afterBox&^s.Candidates[row][col]), candidateList(s.Candidates[row][col])))
	}
	return details
}
