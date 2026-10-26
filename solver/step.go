package solver

// Step describes a change already applied to the board or candidates.
type CandidateElimination struct {
	Row    int `json:"row"`
	Col    int `json:"col"`
	Number int `json:"number"`
}

type Step struct {
	Kind             string      `json:"kind"` // placement or elimination
	Visual           *StepVisual `json:"visual,omitempty"`
	Technique        string      `json:"technique"`
	Explanation      string      `json:"explanation"`
	Details          []string    `json:"details"`
	Row              int         `json:"row"`
	Col              int         `json:"col"`
	Number           int         `json:"number"`
	BeforeCandidates uint16      `json:"beforeCandidates"`

	// Populated by candidate-elimination techniques.
	Eliminations []CandidateElimination `json:"eliminations"`
}
