// Package puzzleapp is the shared API/browser contract for the Go solver.
package puzzleapp

import (
	"fmt"
	"strings"

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
	Focus        *solver.Cell `json:"focus,omitempty"`
	Technique    string       `json:"technique,omitempty"`
	Guidance     string       `json:"guidance,omitempty"`
	UnitGuidance string       `json:"unitGuidance,omitempty"`
	Status       string       `json:"status"`
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
	return guidanceForStep(step), nil
}

func guidanceForStep(step solver.Step) HintResponse {
	result := HintResponse{Status: "hint", Technique: step.Technique}
	units := []string{}
	if step.Visual != nil {
		for _, unit := range step.Visual.Units {
			units = append(units, fmt.Sprintf("%s %d", unit.Kind, unit.Index+1))
		}
	}
	location := strings.Join(units, ", ")
	if step.Number > 0 {
		result.Focus = &solver.Cell{Row: step.Row, Col: step.Col}
		if location == "" {
			location = fmt.Sprintf("row %d, column %d and box %d", step.Row+1, step.Col+1, step.Row/3*3+step.Col/3+1)
		}
		switch step.Technique {
		case "Full House":
			result.Guidance = "Look for a unit with just one empty cell."
			result.UnitGuidance = "Check " + location + ". Which digit is missing?"
		case "Hidden Single":
			result.Guidance = "Look for a digit that fits in only one cell of a unit."
			result.UnitGuidance = "Check " + location + ". Compare where each missing digit can go."
		default:
			result.Guidance = "Look for a cell with only one possible digit."
			result.UnitGuidance = fmt.Sprintf("At row %d, column %d, use %s to rule out digits.", step.Row+1, step.Col+1, location)
		}
	} else {
		result.Guidance = "Look for a " + step.Technique + " pattern."
		if location != "" {
			result.UnitGuidance = "Compare candidates in " + location + ". What can the pattern rule out?"
		} else if step.Visual != nil && len(step.Visual.Sources) > 0 {
			cell := step.Visual.Sources[0]
			result.UnitGuidance = fmt.Sprintf("Start at row %d, column %d. Compare candidates in its row, column and box.", cell.Row+1, cell.Col+1)
		} else {
			result.UnitGuidance = "Compare candidate positions in the rows, columns and boxes involved in the pattern."
		}
	}
	return result
}
