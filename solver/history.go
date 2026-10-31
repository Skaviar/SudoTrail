package solver

// BoardState is a snapshot of the Sudoku board and its candidates.
type BoardState struct {
	initialBoard [9][9]int8
	initialKnown bool
	pending      candidateUpdate
	Board        [9][9]int8
	Candidates   [9][9]uint16
}

// CaptureBoardState copies the current solver state.
// Arrays are copied by value, so later solver changes won't affect the snapshot.
func CaptureBoardState(s *Sudoku) BoardState {
	return BoardState{
		initialBoard: s.initialBoard, initialKnown: s.initialKnown,
		pending:    s.pending,
		Board:      s.Board,
		Candidates: s.Candidates,
	}
}

// Restore copies a saved snapshot back into a Sudoku solver.
func (state BoardState) Restore(s *Sudoku) {
	// The uniqueness cache belongs to these original clues, not the current
	// candidates. Discard it when restoring a different puzzle.
	if s.initialKnown != state.initialKnown || s.initialBoard != state.initialBoard {
		s.uniqueness = 0
	}
	s.initialBoard, s.initialKnown = state.initialBoard, state.initialKnown
	s.pending = state.pending
	s.deferPeerUpdates = false
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
	if step.Visual != nil {
		visual := *step.Visual
		visual.Inferences = append([]InferenceLink(nil), visual.Inferences...)
		visual.Proofs = append([]ProofBranch(nil), visual.Proofs...)
		for i := range visual.Proofs {
			visual.Proofs[i].Nodes = append([]ProofNode(nil), visual.Proofs[i].Nodes...)
			visual.Proofs[i].Conflict = append([]int(nil), visual.Proofs[i].Conflict...)
			for j := range visual.Proofs[i].Nodes {
				visual.Proofs[i].Nodes[j].Parents = append([]int(nil), visual.Proofs[i].Nodes[j].Parents...)
			}
		}
		visual.Highlights = append([]CandidateHighlight(nil), visual.Highlights...)
		visual.CandidateLinks = append([]CandidateLink(nil), visual.CandidateLinks...)
		visual.Links = append([]VisualLink(nil), visual.Links...)
		visual.Restrictions = append([]CandidateElimination(nil), visual.Restrictions...)
		visual.Sources = append([]Cell(nil), visual.Sources...)
		visual.Digits = append([]int(nil), visual.Digits...)
		visual.Units = append([]VisualUnit(nil), visual.Units...)
		step.Visual = &visual
	}
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
