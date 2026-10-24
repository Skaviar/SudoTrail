package solver

// BoardState is a snapshot of the Sudoku board and its candidates.
type BoardState struct {
	Board      [9][9]int8
	Candidates [9][9]uint16
}

// CaptureBoardState copies the current solver state.
// Arrays are copied by value, so later solver changes won't affect the snapshot.
func CaptureBoardState(s *Sudoku) BoardState {
	return BoardState{
		Board:      s.Board,
		Candidates: s.Candidates,
	}
}

// Restore copies a saved snapshot back into a Sudoku solver.
func (state BoardState) Restore(s *Sudoku) {
	s.Board = state.Board
	s.Candidates = state.Candidates
}

// HistoryEntry stores a state and the step that produced it.
// The initial puzzle has HasStep == false.
type HistoryEntry struct {
	State   BoardState
	Step    Step
	HasStep bool
}

// SolverHistory stores snapshots in solving order.
type SolverHistory struct {
	Entries []HistoryEntry
}

// NewSolverHistory records the initial puzzle as entry 0.
func NewSolverHistory(s *Sudoku) *SolverHistory {
	return &SolverHistory{
		Entries: []HistoryEntry{
			{
				State: CaptureBoardState(s),
			},
		},
	}
}

// RecordStep stores the state after a step has been applied.
// The first entry is the original puzzle; subsequent entries follow solving order.
func (h *SolverHistory) RecordStep(s *Sudoku, step Step) {
	step.Details = append([]string(nil), step.Details...)

	// Copy the slice so the history doesn't share its backing array.
	step.Eliminations = append(
		[]CandidateElimination(nil),
		step.Eliminations...,
	)

	h.Entries = append(h.Entries, HistoryEntry{
		State:   CaptureBoardState(s),
		Step:    step,
		HasStep: true,
	})

}

// Len returns the number of recorded states.
func (h *SolverHistory) Len() int { return len(h.Entries) }
