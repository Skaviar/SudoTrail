package solver

import "fmt"

func FindNakedSingle(s *Sudoku) (Step, bool) {
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
