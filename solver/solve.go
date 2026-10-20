package solver

func (s *Sudoku) SolveStep() (Step, bool) {
	if step, ok := FindFullHouse(s); ok {
		return step, true
	}

	if step, ok := FindHiddenSingle(s); ok {
		return step, true
	}

	if step, ok := FindNakedSingle(s); ok {
		return step, true
	}

	if step, ok := FindPointing(s); ok {
		return step, true
	}

	if step, ok := FindClaiming(s); ok {
		return step, true
	}

	if step, ok := FindHiddenSubset(s, 2); ok {
		return step, true
	}
	if step, ok := FindHiddenSubset(s, 3); ok {
		return step, true
	}
	if step, ok := FindHiddenSubset(s, 4); ok {
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

		PrintStep(step, stepNumber, s)

		WaitForEnter()

		stepNumber++
	}
}

// Benchmark mode: no printing and no pauses.
func (s *Sudoku) SolveAutomatically() {
	for {
		_, ok := s.SolveStep()
		if !ok {
			return
		}
	}
}
