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

			details := []string{
				fmt.Sprintf(
					"R%dC%d has candidates: %v.",
					row+1,
					col+1,
					candidateList(candidates),
				),
				"The row, column, and 3×3 box have eliminated all other numbers.",
				fmt.Sprintf(
					"Only %d remains as a valid candidate.",
					number,
				),
				fmt.Sprintf(
					"Therefore R%dC%d = %d.",
					row+1,
					col+1,
					number,
				),
			}

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
