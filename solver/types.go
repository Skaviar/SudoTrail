package solver

import (
	"fmt"
	"math/bits"
)

const AllCandidates uint16 = 0b1111111110 // bits 1-9

type Puzzle struct {
	Id         string
	Puzzle     string
	Solution   string
	Clues      string
	Difficulty string
}

type Sudoku struct {
	Board      [9][9]int8
	Candidates [9][9]uint16
}

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

	// Remove numbers from row
	for c := range 9 {
		value := s.Board[row][c]

		if value != 0 {
			candidates = removeCandidate(candidates, int(value))
		}
	}

	// Remove numbers from column
	for r := range 9 {
		value := s.Board[r][col]

		if value != 0 {
			candidates = removeCandidate(candidates, int(value))
		}
	}

	// Remove numbers from box
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

func hasCandidate(mask uint16, number int) bool {
	return mask&(1<<number) != 0
}

func removeCandidate(mask uint16, number int) uint16 {
	return mask &^ (1 << number)
}

func candidateCount(mask uint16) int {
	return bits.OnesCount16(mask)
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
	// Same row
	for c := range 9 {
		if c != col {
			s.Candidates[row][c] =
				removeCandidate(s.Candidates[row][c], number)
		}
	}

	// Same column
	for r := range 9 {
		if r != row {
			s.Candidates[r][col] =
				removeCandidate(s.Candidates[r][col], number)
		}
	}

	// Same box
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

func (s *Sudoku) Print() {
	fmt.Println("+-------+-------+-------+")

	for row := range 9 {
		fmt.Print("| ")

		for col := range 9 {
			value := s.Board[row][col]

			if value == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", value)
			}

			if col == 2 || col == 5 {
				fmt.Print("| ")
			}
		}

		fmt.Println("|")

		if row == 2 || row == 5 {
			fmt.Println("+-------+-------+-------+")
		}
	}

	fmt.Println("+-------+-------+-------+")
}

// --------------------------------------------------
// Debugging candidates
// --------------------------------------------------

func (s *Sudoku) PrintCandidates() {
	for row := range 9 {
		for col := range 9 {
			if s.Board[row][col] != 0 {
				continue
			}

			fmt.Printf(
				"(%d,%d): %v\n",
				row,
				col,
				candidateList(s.Candidates[row][col]),
			)
		}
	}
}

func candidateList(mask uint16) []int {
	var result []int

	for number := 1; number <= 9; number++ {
		if hasCandidate(mask, number) {
			result = append(result, number)
		}
	}

	return result
}
