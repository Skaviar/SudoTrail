package solver

import (
	"fmt"
)

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

type Step struct {
	Technique        string
	Explanation      string
	Row              int
	Col              int
	Number           int
	BeforeCandidates uint16
}

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
				CandidateList(s.Candidates[row][col]),
			)
		}
	}
}
