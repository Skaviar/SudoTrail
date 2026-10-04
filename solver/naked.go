package solver

import (
	"fmt"
)

func FindNakedSingle(s *Sudoku) (Step, bool) {
	for row := range 9 {
		for col := range 9 {
			// Skip already-filled cells.
			if s.Board[row][col] != 0 {
				continue
			}

			candidates := s.Candidates[row][col]

			// Naked Single means exactly one candidate.
			if CandidateCount(candidates) != 1 {
				continue
			}

			number := SingleCandidate(candidates)

			s.Place(row, col, number)

			return Step{
				Technique: "Naked Single",
				Explanation: fmt.Sprintf(
					"Cell R%dC%d has only one possible candidate: %d.",
					row+1,
					col+1,
					number,
				),
				Row:              row,
				Col:              col,
				Number:           number,
				BeforeCandidates: candidates,
			}, true
		}
	}

	return Step{}, false
}
