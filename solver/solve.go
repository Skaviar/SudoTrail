package solver

import "fmt"

func (s *Sudoku) SolveStep() (Step, bool) {
	if step, ok := s.FindFullHouse(); ok {
		return step, true
	}

	if step, ok := s.FindHiddenSingle(); ok {
		return step, true
	}

	if step, ok := s.FindNakedSingle(); ok {
		return step, true
	}

	return Step{}, false
}

func (s *Sudoku) Solve() {
	stepNumber := 1

	for {
		step, ok := s.SolveStep()

		if !ok {
			break
		}

		fmt.Printf("\n================ STEP %d ================\n", stepNumber)
		fmt.Printf("Technique: %s\n", step.Technique)
		fmt.Printf("Explanation: %s\n", step.Explanation)

		s.Print()

		stepNumber++
	}
}
