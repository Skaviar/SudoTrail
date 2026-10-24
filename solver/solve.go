package solver

// SolveStep applies one technique. A false result means no supported move remains.
func (s *Sudoku) SolveStep() (Step, bool) {
	if step, ok := ApplyFullHouse(s); ok {
		return step, true
	}

	if step, ok := ApplyHiddenSingle(s); ok {
		return step, true
	}

	if step, ok := ApplyNakedSingle(s); ok {
		return step, true
	}

	if step, ok := ApplyPointing(s); ok {
		return step, true
	}

	if step, ok := ApplyClaiming(s); ok {
		return step, true
	}

	if step, ok := ApplyHiddenSubset(s, 2); ok {
		return step, true
	}
	if step, ok := ApplyHiddenSubset(s, 3); ok {
		return step, true
	}
	if step, ok := ApplyHiddenSubset(s, 4); ok {
		return step, true
	}

	return Step{}, false
}

// Solve applies available techniques and returns the complete history.
// Inspect IsSolved afterward to distinguish a solved puzzle from a stuck one.
func (s *Sudoku) Solve() *SolverHistory {
	history := NewSolverHistory(s)
	for {
		step, ok := s.SolveStep()
		if !ok {
			return history
		}
		history.RecordStep(s, step)
	}
}

// SolveAutomatically avoids history allocation for database benchmarks.
func (s *Sudoku) SolveAutomatically() {
	for {
		if _, ok := s.SolveStep(); !ok {
			return
		}
	}
}
