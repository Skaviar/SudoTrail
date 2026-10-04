package solver

func (s *Sudoku) SolveStep() bool {
	if s.FindFullHouse() {
		return true
	}

	if s.FindHiddenSingle() {
		return true
	}

	if s.FindNakedSingle() {
		return true
	}

	return false
}

func (s *Sudoku) Solve() {
	for s.SolveStep() {
	}
}
