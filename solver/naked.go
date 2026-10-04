package solver

func (s *Sudoku) FindNakedSingle() bool {
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] != 0 {
				continue
			}

			if candidateCount(s.Candidates[row][col]) == 1 {
				number := singleCandidate(s.Candidates[row][col])

				s.Place(row, col, number)

				return true
			}
		}
	}

	return false
}

func singleCandidate(mask uint16) int {
	if candidateCount(mask) != 1 {
		return 0
	}

	for number := 1; number <= 9; number++ {
		if hasCandidate(mask, number) {
			return number
		}
	}

	return 0
}
