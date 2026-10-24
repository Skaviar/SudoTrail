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
	Current int
}

// NewSolverHistory records the initial puzzle as entry 0.
func NewSolverHistory(s *Sudoku) *SolverHistory {
	return &SolverHistory{
		Entries: []HistoryEntry{
			{
				State: CaptureBoardState(s),
			},
		},
		Current: 0,
	}
}

// RecordStep stores the state after a step has been applied.
// If we're branching from an earlier state, discard the old future.
func (h *SolverHistory) RecordStep(s *Sudoku, step Step) {
	if h.Current < len(h.Entries)-1 {
		h.Entries = h.Entries[:h.Current+1]
	}

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

	h.Current = len(h.Entries) - 1
}

// Previous moves back one recorded state.
func (h *SolverHistory) Previous() {
	if h.Current > 0 {
		h.Current--
	}
}

// Next moves forward one recorded state.
func (h *SolverHistory) Next() {
	if h.Current < len(h.Entries)-1 {
		h.Current++
	}
}

// First jumps to the initial puzzle.
func (h *SolverHistory) First() {
	h.Current = 0
}

// Last jumps to the latest recorded state.
func (h *SolverHistory) Last() {
	if len(h.Entries) > 0 {
		h.Current = len(h.Entries) - 1
	}
}

// CurrentEntry returns the currently selected history entry.
func (h *SolverHistory) CurrentEntry() (HistoryEntry, bool) {
	if len(h.Entries) == 0 ||
		h.Current < 0 ||
		h.Current >= len(h.Entries) {
		return HistoryEntry{}, false
	}

	return h.Entries[h.Current], true
}

// RestoreCurrent restores the selected snapshot into the solver.
func (h *SolverHistory) RestoreCurrent(s *Sudoku) bool {
	entry, ok := h.CurrentEntry()
	if !ok {
		return false
	}

	entry.State.Restore(s)
	return true
}

// Len returns the number of recorded states.
func (h *SolverHistory) Len() int {
	return len(h.Entries)
}
