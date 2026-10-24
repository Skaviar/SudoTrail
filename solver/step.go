package solver

// Step describes a change already applied to the board or candidates.
type CandidateElimination struct {
	Row    int
	Col    int
	Number int
}

type Step struct {
	Technique        string
	Explanation      string
	Details          []string
	Row              int
	Col              int
	Number           int
	BeforeCandidates uint16

	// Populated by candidate-elimination techniques.
	Eliminations []CandidateElimination
}
