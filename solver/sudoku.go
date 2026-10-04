package solver

func (s *Sudoku) InitializeSudoku(p Puzzle) {
	for row := range 9 {
		for col := range 9 {
			index := row*9 + col

			if p.Puzzle[index] == '.' {
				s.Board[row][col] = 0
			} else {
				s.Board[row][col] = int8(p.Puzzle[index] - '0')
			}
		}
	}

	s.initializeCandidates()
}

func (s *Sudoku) initializeCandidates() {
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] != 0 {
				s.Candidates[row][col] = 0
				continue
			}

			s.Candidates[row][col] = s.calculateCandidates(row, col)
		}
	}
}

func (s *Sudoku) calculateCandidates(row, col int) uint16 {
	var candidates uint16 = AllCandidates

	// Row
	for c := range 9 {
		value := s.Board[row][c]

		if value != 0 {
			candidates = removeCandidate(candidates, int(value))
		}
	}

	// Column
	for r := range 9 {
		value := s.Board[r][col]

		if value != 0 {
			candidates = removeCandidate(candidates, int(value))
		}
	}

	// Box
	boxRow := (row / 3) * 3
	boxCol := (col / 3) * 3

	for r := boxRow; r < boxRow+3; r++ {
		for c := boxCol; c < boxCol+3; c++ {
			value := s.Board[r][c]

			if value != 0 {
				candidates = removeCandidate(candidates, int(value))
			}
		}
	}

	return candidates
}

func (s *Sudoku) Place(row, col, number int) {
	if number < 1 || number > 9 {
		return
	}

	s.Board[row][col] = int8(number)
	s.Candidates[row][col] = 0

	s.updateCandidates(row, col, number)
}

func (s *Sudoku) updateCandidates(row, col, number int) {
	// Row
	for c := range 9 {
		if c != col {
			s.Candidates[row][c] =
				removeCandidate(s.Candidates[row][c], number)
		}
	}

	// Column
	for r := range 9 {
		if r != row {
			s.Candidates[r][col] =
				removeCandidate(s.Candidates[r][col], number)
		}
	}

	// Box
	boxRow := (row / 3) * 3
	boxCol := (col / 3) * 3

	for r := boxRow; r < boxRow+3; r++ {
		for c := boxCol; c < boxCol+3; c++ {
			if r != row || c != col {
				s.Candidates[r][c] =
					removeCandidate(s.Candidates[r][c], number)
			}
		}
	}
}
