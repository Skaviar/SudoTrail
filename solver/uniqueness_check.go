package solver

// countSolutions uses backtracking only to distinguish no solution, one
// solution, and multiple solutions. It stops at two and never returns a
// completed board to the logical solver.
func countSolutions(board [9][9]int8) int {
	var rows, columns, boxes [9]uint16
	for row := range 9 {
		for col := range 9 {
			number := board[row][col]
			if number == 0 {
				continue
			}
			if number < 1 || number > 9 {
				return 0
			}
			bit, box := uint16(1)<<number, row/3*3+col/3
			if (rows[row]|columns[col]|boxes[box])&bit != 0 {
				return 0
			}
			rows[row] |= bit
			columns[col] |= bit
			boxes[box] |= bit
		}
	}
	count := 0
	var visit func()
	visit = func() {
		if count >= 2 {
			return
		}
		best := Cell{Row: -1}
		bestCount := 10
		var choices uint16
		for row := range 9 {
			for col := range 9 {
				if board[row][col] != 0 {
					continue
				}
				mask := AllCandidates &^ (rows[row] | columns[col] | boxes[row/3*3+col/3])
				possible := candidateCount(mask)
				if possible == 0 {
					return
				}
				if possible < bestCount {
					best, bestCount, choices = Cell{row, col}, possible, mask
				}
			}
		}
		if best.Row == -1 {
			count++
			return
		}
		row, col := best.Row, best.Col
		box := row/3*3 + col/3
		for number := 1; number <= 9 && count < 2; number++ {
			bit := uint16(1) << number
			if choices&bit == 0 {
				continue
			}
			board[row][col] = int8(number)
			rows[row] |= bit
			columns[col] |= bit
			boxes[box] |= bit
			visit()
			board[row][col] = 0
			rows[row] &^= bit
			columns[col] &^= bit
			boxes[box] &^= bit
		}
	}
	visit()
	return count
}

// HasUniqueSolution verifies the original submitted clues, ignoring deductions
// and later placements. Only this yes/no result is used by uniqueness methods;
// backtracking never places numbers or eliminates candidates in the puzzle.
func (s *Sudoku) HasUniqueSolution() bool {
	if s.initialKnown && s.uniqueness != 0 {
		return s.uniqueness == 1
	}
	clues := s.Board
	if s.initialKnown {
		clues = s.initialBoard
	}
	unique := countSolutions(clues) == 1
	if s.initialKnown {
		s.uniqueness = -1
		if unique {
			s.uniqueness = 1
		}
	}
	return unique
}
