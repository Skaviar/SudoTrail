package solver

// SolveStep separates placement from its peer-candidate cleanup. The pending
// cleanup always runs before another technique can inspect the candidates.
func (s *Sudoku) SolveStep() (Step, bool) {
	if step, ok := s.applyPendingCandidates(); ok {
		return step, true
	}
	s.deferPeerUpdates = true
	defer func() { s.deferPeerUpdates = false }()
	step, ok := s.applyTechnique()
	if ok {
		if step.Number > 0 {
			step.Kind = "placement"
		} else {
			step.Kind = "elimination"
		}
	}
	return step, ok
}

func (s *Sudoku) applyTechnique() (Step, bool) {
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

	// Try smaller subsets before larger ones, keeping the simpler deduction first.
	if step, ok := ApplyNakedSubset(s, 2); ok {
		return step, true
	}
	if step, ok := ApplyHiddenSubset(s, 2); ok {
		return step, true
	}
	if step, ok := ApplyNakedSubset(s, 3); ok {
		return step, true
	}
	if step, ok := ApplyHiddenSubset(s, 3); ok {
		return step, true
	}
	if step, ok := ApplyNakedSubset(s, 4); ok {
		return step, true
	}
	if step, ok := ApplyHiddenSubset(s, 4); ok {
		return step, true
	}
	if step, ok := ApplyBasicFish(s, 2); ok {
		return step, true
	}
	if step, ok := ApplySkyscraper(s); ok {
		return step, true
	}
	if step, ok := ApplyTwoStringKite(s); ok {
		return step, true
	}
	if step, ok := ApplyTurbotFish(s); ok {
		return step, true
	}
	if step, ok := ApplyEmptyRectangle(s); ok {
		return step, true
	}
	if step, ok := ApplyXYWing(s); ok {
		return step, true
	}
	if step, ok := ApplyXYZWing(s); ok {
		return step, true
	}
	if step, ok := ApplyWWing(s); ok {
		return step, true
	}
	if step, ok := ApplyBasicFish(s, 3); ok {
		return step, true
	}
	if step, ok := ApplyBasicFish(s, 4); ok {
		return step, true
	}
	if step, ok := ApplyFinnedXWing(s); ok {
		return step, true
	}
	if step, ok := ApplyRemotePairs(s); ok {
		return step, true
	}
	if step, ok := ApplySimpleColoring(s); ok {
		return step, true
	}
	if step, ok := ApplyUniqueRectangle(s, 1); ok {
		return step, true
	}
	if step, ok := ApplyUniqueRectangle(s, 2); ok {
		return step, true
	}
	if step, ok := ApplyBUGPlusOne(s); ok {
		return step, true
	}
	for size := 5; size <= 7; size++ {
		if step, ok := ApplyBasicFish(s, size); ok {
			return step, true
		}
	}
	for size := 3; size <= 7; size++ {
		if step, ok := ApplyFinnedFish(s, size); ok {
			return step, true
		}
	}
	return s.applyAdvancedTechniques()
}

// The advanced stage is reached only when the simpler named patterns stall.
// Graphs and ALSs belong to this candidate state and are reused within it.
func (s *Sudoku) applyAdvancedTechniques() (Step, bool) {
	if s.IsSolved() {
		return Step{}, false
	}
	if step, ok := ApplyAdvancedRectangles(s); ok {
		return step, true
	}
	if step, ok := ApplyMissingCandidateRectangles(s); ok {
		return step, true
	}
	if step, ok := ApplyAvoidableRectangle(s); ok {
		return step, true
	}
	g := buildCandidateGraph(s)
	if step, ok := applyMultiColoring(s, g); ok {
		return step, true
	}
	if step, ok := applyMedusa(s, g); ok {
		return step, true
	}
	for _, mode := range []string{"X-Chain", "XY-Chain", "Alternating Inference Chain"} {
		if step, ok := applyChains(s, g, mode); ok {
			return step, true
		}
	}
	als := buildALS(s)
	if step, ok := applyALS(s, als); ok {
		return step, true
	}
	if step, ok := applyDeathBlossom(s, als); ok {
		return step, true
	}
	for size := 2; size <= 7; size++ {
		if step, ok := ApplySiameseFish(s, size); ok {
			return step, true
		}
		if step, ok := ApplyComplexFish(s, size); ok {
			return step, true
		}
	}
	if step, ok := applyForcing(s, g, false); ok {
		return step, true
	}
	return applyForcing(s, g, true)
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
	// Benchmarks do not need separate teaching steps.
	s.applyPendingCandidates()
	for {
		if _, ok := s.applyTechnique(); !ok {
			return
		}
	}
}
