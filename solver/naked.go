package solver

import "fmt"

func ApplyNakedSingle(s *Sudoku) (Step, bool) {
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

			visual := singleCellVisual(s, Cell{Row: row, Col: col}, number)
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
				Visual:           visual,
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

func explainCellCandidates(s *Sudoku, row, col int) []string {
	target := Cell{Row: row, Col: col}
	number := singleCandidate(s.Candidates[row][col])
	visual := singleCellVisual(s, target, number)
	var details []string
	for _, link := range visual.Links {
		details = append(details, fmt.Sprintf("%s contains %d and shares %s with %s, so %d is not possible here.", cellName(link.From.Row, link.From.Col), link.Number, unitLabel(link.Unit), cellName(row, col), link.Number))
	}
	earlier := s.calculateCandidates(row, col) &^ s.Candidates[row][col]
	if earlier != 0 {
		details = append(details, fmt.Sprintf("Earlier solving steps ruled out %v in this cell.", candidateList(earlier)))
	}
	return append(details, fmt.Sprintf("Remaining candidates for %s: %v.", cellName(row, col), candidateList(s.Candidates[row][col])))
}
