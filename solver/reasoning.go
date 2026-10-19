package solver

import (
	"fmt"
	"strings"
)

// --------------------------------------------------
// Unit numbers
// --------------------------------------------------

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
	details := []string{
		fmt.Sprintf(
			"We are looking for %d in Row %d.",
			number,
			row+1,
		),
	}

	details = append(
		details,
		explainHiddenSingleRowEliminations(s, row, number)...,
	)

	possible := hiddenSinglePossibleCellsRow(
		s,
		row,
		number,
	)

	if len(possible) == 1 {
		details = append(
			details,
			fmt.Sprintf(
				"Only R%dC%d remains.",
				row+1,
				possible[0]+1,
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
	details := []string{
		fmt.Sprintf(
			"We are looking for %d in Column %d.",
			number,
			col+1,
		),
	}

	details = append(
		details,
		explainHiddenSingleColumnEliminations(s, col, number)...,
	)

	possible := hiddenSinglePossibleCellsColumn(
		s,
		col,
		number,
	)

	if len(possible) == 1 {
		details = append(
			details,
			fmt.Sprintf(
				"Only R%dC%d remains.",
				possible[0]+1,
				col+1,
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
	box := boxNumber(boxRow, boxCol)

	details := []string{
		fmt.Sprintf(
			"We are looking for %d in Box %d.",
			number,
			box,
		),
	}

	details = append(
		details,
		explainHiddenSingleBoxEliminations(
			s,
			boxRow,
			boxCol,
			number,
		)...,
	)

	possible := hiddenSinglePossibleCellsBox(
		s,
		boxRow,
		boxCol,
		number,
	)

	if len(possible) == 1 {
		details = append(
			details,
			fmt.Sprintf(
				"Only R%dC%d remains.",
				possible[0][0]+1,
				possible[0][1]+1,
			),
		)
	}

	return details
}

// --------------------------------------------------
// Hidden Single - Row
// --------------------------------------------------

func explainHiddenSingleRowEliminations(
	s *Sudoku,
	row int,
	number int,
) []string {
	/*
		When looking for a number in a row:

		Each empty cell can be eliminated because of:
		- its column
		- its box

		We group cells by the reason.

		Example:

		4 is already in Columns 2, 3, 4, and 8
		→ eliminate R6C2, R6C3, R6C4, and R6C8.

		4 is already in Box 5
		→ eliminate R6C5 and R6C6.
	*/

	var columnCells []string
	var columnNumbers []int

	var boxCells []string
	var boxNumbers []int

	for col := range 9 {
		if s.Board[row][col] != 0 {
			continue
		}

		if hasCandidate(s.Candidates[row][col], number) {
			continue
		}

		reason, unit := hiddenSingleElimination(
			s,
			row,
			col,
			number,
		)

		switch reason {
		case "column":
			columnNumbers = appendUnique(columnNumbers, unit)
			columnCells = append(
				columnCells,
				cellName(row, col),
			)

		case "box":
			boxNumbers = appendUnique(boxNumbers, unit)
			boxCells = append(
				boxCells,
				cellName(row, col),
			)
		}
	}

	var details []string

	if len(columnCells) > 0 {
		details = append(
			details,
			fmt.Sprintf(
				"%d is already in %s → eliminate %s.",
				number,
				formatUnits("Column", columnNumbers),
				joinCells(columnCells),
			),
		)
	}

	if len(boxCells) > 0 {
		details = append(
			details,
			fmt.Sprintf(
				"%d is already in %s → eliminate %s.",
				number,
				formatUnits("Box", boxNumbers),
				joinCells(boxCells),
			),
		)
	}

	return details
}

// --------------------------------------------------
// Hidden Single - Column
// --------------------------------------------------

func explainHiddenSingleColumnEliminations(
	s *Sudoku,
	col int,
	number int,
) []string {
	var rowCells []string
	var rowNumbers []int

	var boxCells []string
	var boxNumbers []int

	for row := range 9 {
		if s.Board[row][col] != 0 {
			continue
		}

		if hasCandidate(s.Candidates[row][col], number) {
			continue
		}

		reason, unit := hiddenSingleElimination(
			s,
			row,
			col,
			number,
		)

		switch reason {
		case "row":
			rowNumbers = appendUnique(rowNumbers, unit)
			rowCells = append(
				rowCells,
				cellName(row, col),
			)

		case "box":
			boxNumbers = appendUnique(boxNumbers, unit)
			boxCells = append(
				boxCells,
				cellName(row, col),
			)
		}
	}

	var details []string

	if len(rowCells) > 0 {
		details = append(
			details,
			fmt.Sprintf(
				"%d is already in %s → eliminate %s.",
				number,
				formatUnits("Row", rowNumbers),
				joinCells(rowCells),
			),
		)
	}

	if len(boxCells) > 0 {
		details = append(
			details,
			fmt.Sprintf(
				"%d is already in %s → eliminate %s.",
				number,
				formatUnits("Box", boxNumbers),
				joinCells(boxCells),
			),
		)
	}

	return details
}

// --------------------------------------------------
// Hidden Single - Box
// --------------------------------------------------

func explainHiddenSingleBoxEliminations(
	s *Sudoku,
	boxRow int,
	boxCol int,
	number int,
) []string {
	var rowCells []string
	var rowNumbers []int

	var columnCells []string
	var columnNumbers []int

	for row := boxRow; row < boxRow+3; row++ {
		for col := boxCol; col < boxCol+3; col++ {
			if s.Board[row][col] != 0 {
				continue
			}

			if hasCandidate(s.Candidates[row][col], number) {
				continue
			}

			reason, unit := hiddenSingleElimination(
				s,
				row,
				col,
				number,
			)

			switch reason {
			case "row":
				rowNumbers = appendUnique(rowNumbers, unit)
				rowCells = append(
					rowCells,
					cellName(row, col),
				)

			case "column":
				columnNumbers = appendUnique(columnNumbers, unit)
				columnCells = append(
					columnCells,
					cellName(row, col),
				)
			}
		}
	}

	var details []string

	if len(rowCells) > 0 {
		details = append(
			details,
			fmt.Sprintf(
				"%d is already in %s → eliminate %s.",
				number,
				formatUnits("Row", rowNumbers),
				joinCells(rowCells),
			),
		)
	}

	if len(columnCells) > 0 {
		details = append(
			details,
			fmt.Sprintf(
				"%d is already in %s → eliminate %s.",
				number,
				formatUnits("Column", columnNumbers),
				joinCells(columnCells),
			),
		)
	}

	return details
}

// --------------------------------------------------
// Determine why a candidate is eliminated
// --------------------------------------------------

func hiddenSingleElimination(
	s *Sudoku,
	row int,
	col int,
	number int,
) (string, int) {

	// Row
	for c := range 9 {
		if c == col {
			continue
		}

		if int(s.Board[row][c]) == number {
			return "row", row + 1
		}
	}

	// Column
	for r := range 9 {
		if r == row {
			continue
		}

		if int(s.Board[r][col]) == number {
			return "column", col + 1
		}
	}

	// Box
	boxRow := (row / 3) * 3
	boxCol := (col / 3) * 3

	for r := boxRow; r < boxRow+3; r++ {
		for c := boxCol; c < boxCol+3; c++ {
			if r == row && c == col {
				continue
			}

			if int(s.Board[r][c]) == number {
				return "box", boxNumber(row, col)
			}
		}
	}

	return "", 0
}

// --------------------------------------------------
// Hidden Single possible cells
// --------------------------------------------------

func hiddenSinglePossibleCellsRow(
	s *Sudoku,
	row int,
	number int,
) []int {
	var cells []int

	for col := range 9 {
		if s.Board[row][col] == 0 &&
			hasCandidate(s.Candidates[row][col], number) {
			cells = append(cells, col)
		}
	}

	return cells
}

func hiddenSinglePossibleCellsColumn(
	s *Sudoku,
	col int,
	number int,
) []int {
	var cells []int

	for row := range 9 {
		if s.Board[row][col] == 0 &&
			hasCandidate(s.Candidates[row][col], number) {
			cells = append(cells, row)
		}
	}

	return cells
}

func hiddenSinglePossibleCellsBox(
	s *Sudoku,
	boxRow int,
	boxCol int,
	number int,
) [][2]int {
	var cells [][2]int

	for row := boxRow; row < boxRow+3; row++ {
		for col := boxCol; col < boxCol+3; col++ {
			if s.Board[row][col] == 0 &&
				hasCandidate(s.Candidates[row][col], number) {
				cells = append(
					cells,
					[2]int{row, col},
				)
			}
		}
	}

	return cells
}

// --------------------------------------------------
// Formatting helpers
// --------------------------------------------------

func cellName(row, col int) string {
	return fmt.Sprintf(
		"R%dC%d",
		row+1,
		col+1,
	)
}

func joinCells(cells []string) string {
	return formatList(cells)
}

func formatUnits(prefix string, units []int) string {
    if len(units) == 1 {
        return fmt.Sprintf("%s %d", prefix, units[0])
    }

    plural := map[string]string{
        "Row":    "Rows",
        "Column": "Columns",
        "Box":    "Boxes",
    }[prefix]

    if plural == "" {
        plural = prefix + "s"
    }

    values := make([]string, len(units))
    for i, unit := range units {
        values[i] = fmt.Sprintf("%d", unit)
    }

    return fmt.Sprintf("%s %s", plural, formatList(values))
}

func formatList(values []string) string {
	switch len(values) {
	case 0:
		return ""

	case 1:
		return values[0]

	case 2:
		return values[0] + " and " + values[1]
	}

	return strings.Join(values[:len(values)-1], ", ") +
		", and " +
		values[len(values)-1]
}

func appendUnique(
	values []int,
	value int,
) []int {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	return append(values, value)
}

func boxNumber(row, col int) int {
	return (row/3)*3 + (col / 3) + 1
}
