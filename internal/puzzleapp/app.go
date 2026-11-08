// Package puzzleapp is the shared API/browser contract for the Go solver.
package puzzleapp

import (
	"fmt"
	"github.com/s-threepio/godoku/solver"
)

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

// HintResponse contains guidance only; solution digits and deductions stay private.
type HintResponse struct {
	Focus     *solver.Cell `json:"focus,omitempty"`
	Technique string       `json:"technique,omitempty"`
	Guidance  string       `json:"guidance,omitempty"`
	Status    string       `json:"status"`
}

func Hint(puzzle string) (HintResponse, error) {
	s, err := solver.NewSudoku(solver.Puzzle{Puzzle: puzzle})
	if err != nil {
		return HintResponse{}, err
	}
	if s.IsSolved() {
		return HintResponse{Status: "solved"}, nil
	}
	step, ok := s.SolvePracticeStep()
	if !ok {
		return HintResponse{Status: "stuck"}, nil
	}
	result := HintResponse{Status: "hint", Technique: step.Technique}
	if step.Number > 0 {
		result.Focus = &solver.Cell{Row: step.Row, Col: step.Col}
		location := fmt.Sprintf("row %d, column %d", step.Row+1, step.Col+1)
		switch step.Technique {
		case "Full House":
			result.Guidance = "Look at " + location + ". Check its row, column and box for a unit with one empty cell, then work out which digit is missing."
		case "Hidden Single":
			result.Guidance = "Look around " + location + ". Check each missing digit in its row, column and box. Can any digit fit in only one cell?"
		default:
			result.Guidance = "Focus on " + location + ". Rule out digits already present in its row, column and box. What possibilities remain?"
		}
	} else {
		result.Guidance = "Look for a " + step.Technique + " pattern. Compare candidate positions across rows, columns and boxes to find what can be ruled out."
	}
	return result, nil
}
