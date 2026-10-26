package solver

import "fmt"

// NewSudoku validates clues before building candidates. Both 0 and . mean empty.
func NewSudoku(p Puzzle) (*Sudoku, error) {
	if issues := ValidatePuzzle(p.Puzzle); len(issues) > 0 {
		return nil, fmt.Errorf("%s", issues[0].Message)
	}
	s := &Sudoku{}
	for index, ch := range []byte(p.Puzzle) {
		if ch == '.' || ch == '0' {
			continue
		}
		s.Board[index/9][index%9] = int8(ch - '0')
	}
	s.initializeCandidates()
	return s, nil
}

// IsSolved verifies that every unit contains all nine digits.
func (s *Sudoku) IsSolved() bool {
	for _, unit := range allUnits() {
		var digits uint16
		for _, cell := range unit.cells {
			number := int(s.Board[cell.Row][cell.Col])
			if number < 1 || number > 9 {
				return false
			}
			digits |= 1 << number
		}
		if digits != AllCandidates {
			return false
		}
	}
	return true
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

	if s.deferPeerUpdates {
		s.pending = s.peerUpdate(row, col, number)
	} else {
		s.updateCandidates(row, col, number)
	}
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
