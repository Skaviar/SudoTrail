// Package puzzleapp is the shared API/browser contract for the Go solver.
package puzzleapp

import "github.com/s-threepio/godoku/solver"

type ValidationResponse struct {
	Valid  bool                     `json:"valid"`
	Issues []solver.ValidationIssue `json:"issues"`
}
type StateResponse struct {
	Board      [9][9]int8   `json:"board"`
	Candidates [9][9]uint16 `json:"candidates"`
	Step       *solver.Step `json:"step"`
}
type SolveResponse struct {
	Solved bool            `json:"solved"`
	Status string          `json:"status"`
	States []StateResponse `json:"states"`
}

func Validate(puzzle string) ValidationResponse {
	issues := solver.ValidatePuzzle(puzzle)
	return ValidationResponse{Valid: len(issues) == 0, Issues: issues}
}
func Solve(puzzle string) (SolveResponse, error) {
	s, err := solver.NewSudoku(solver.Puzzle{Puzzle: puzzle})
	if err != nil {
		return SolveResponse{}, err
	}
	// Match the catalog's difficulty order so a promised practice technique
	// really occurs in the history shown to the learner.
	history := s.SolvePractice()
	result := SolveResponse{Solved: s.IsSolved(), Status: "stuck", States: make([]StateResponse, 0, history.Len())}
	if result.Solved {
		result.Status = "solved"
	}
	for _, entry := range history.Entries {
		state := StateResponse{Board: entry.State.Board, Candidates: entry.State.Candidates}
		if entry.HasStep {
			step := entry.Step
			state.Step = &step
		}
		result.States = append(result.States, state)
	}
	return result, nil
}
