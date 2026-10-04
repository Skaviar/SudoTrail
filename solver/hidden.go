package solver

func (s *Sudoku) FindHiddenSingle() bool {
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
				return true
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
				return true
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
				return true
			}
		}
	}

	return false
}
