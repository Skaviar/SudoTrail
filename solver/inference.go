package solver

import "fmt"

// A candidate is a proposition such as "R3C4 contains 7". Strong links say
// at least one endpoint is true; weak links say at most one is true.
type Candidate struct {
	Row    int `json:"row"`
	Col    int `json:"col"`
	Number int `json:"number"`
}
type CandidateLiteral struct {
	Candidate
	True bool `json:"true"`
}
type InferenceLink struct {
	From   Candidate `json:"from"`
	To     Candidate `json:"to"`
	Kind   string    `json:"kind"`
	Reason string    `json:"reason"`
}
type ProofNode struct {
	CandidateLiteral
	Parents []int  `json:"parents,omitempty"` // Earlier nodes in this branch.
	Reason  string `json:"reason"`
}
type ProofBranch struct {
	Nodes         []ProofNode `json:"nodes"`
	Contradiction string      `json:"contradiction,omitempty"`
	Conflict      []int       `json:"conflict,omitempty"`
}

func candidateID(i, d int) int { return i*9 + d - 1 }
func candidateAt(id int) Candidate {
	return Candidate{Row: id / 9 / 9, Col: id / 9 % 9, Number: id%9 + 1}
}
func (c Candidate) cell() Cell     { return Cell{c.Row, c.Col} }
func (c Candidate) String() string { return fmt.Sprintf("%s=%d", cellName(c.Row, c.Col), c.Number) }
func literalAt(lit int) CandidateLiteral {
	return CandidateLiteral{Candidate: candidateAt(lit / 2), True: lit%2 == 1}
}
func literalText(lit int) string {
	c := candidateAt(lit / 2)
	if lit%2 == 1 {
		return c.String()
	}
	return fmt.Sprintf("%s is not %d", cellName(c.Row, c.Col), c.Number)
}
func candidatesConflict(a, b int) bool {
	return a != b && (a/9 == b/9 || a%9 == b%9 && topology.peers[a/9].has(b/9))
}

type inferenceEdge struct {
	to     int
	reason string
	cell   bool
}
type candidateGraph struct {
	active          []int
	constraints     [][]int // Exactly one candidate in each cell or unit/digit.
	constraintNames []string
	memberships     [729][]int
	edges           [1458][]inferenceEdge
}

func buildCandidateGraph(s *Sudoku) *candidateGraph {
	g := &candidateGraph{}
	addConstraint := func(ids []int, reason string, cell bool) {
		if len(ids) == 0 {
			return
		}
		u := len(g.constraints)
		g.constraints = append(g.constraints, ids)
		g.constraintNames = append(g.constraintNames, reason)
		for _, a := range ids {
			g.memberships[a] = append(g.memberships[a], u)
			for _, b := range ids {
				if a == b {
					continue
				}
				g.edges[a*2+1] = append(g.edges[a*2+1], inferenceEdge{b * 2, reason, cell})
				if len(ids) == 2 {
					g.edges[a*2] = append(g.edges[a*2], inferenceEdge{b*2 + 1, reason, cell})
				}
			}
		}
	}
	for i := range 81 {
		c := cellIndex(i)
		if s.Board[c.Row][c.Col] != 0 {
			continue
		}
		var ids []int
		for _, d := range candidateList(s.Candidates[c.Row][c.Col]) {
			id := candidateID(i, d)
			ids = append(ids, id)
			g.active = append(g.active, id)
		}
		addConstraint(ids, cellName(c.Row, c.Col), true)
	}
	for u, house := range allUnits() {
		for d := 1; d <= 9; d++ {
			var ids []int
			for _, c := range candidateCells(s, house, d) {
				ids = append(ids, candidateID(indexCell(c), d))
			}
			addConstraint(ids, fmt.Sprintf("%d in %s", d, unitLabel(visualHouse(u))), false)
		}
	}
	return g
}

func addInference(v *StepVisual, from, to int, reason string) {
	kind := "weak"
	if from%2 == 0 {
		kind = "strong"
	}
	v.Inferences = append(v.Inferences, InferenceLink{candidateAt(from / 2), candidateAt(to / 2), kind, reason})
	for _, lit := range []int{from, to} {
		c := candidateAt(lit / 2)
		if !containsCell(v.Sources, c.cell()) {
			v.Sources = append(v.Sources, c.cell())
		}
		role := "color-a"
		if lit%2 == 0 {
			role = "color-b"
		}
		v.Highlights = append(v.Highlights, CandidateHighlight{c.Row, c.Col, c.Number, role})
	}
}

// commitEliminations is used after a complete proof has been built. A rejected
// or empty deduction never changes the puzzle.
func commitEliminations(s *Sudoku, name, explanation string, targets []CandidateElimination, details []string, v *StepVisual) (Step, bool) {
	var removed []CandidateElimination
	for _, e := range targets {
		c := Cell{e.Row, e.Col}
		if !cellHasCandidate(s, c, e.Number) {
			continue
		}
		s.Candidates[e.Row][e.Col] = removeCandidate(s.Candidates[e.Row][e.Col], e.Number)
		removed = append(removed, e)
	}
	if len(removed) == 0 {
		return Step{}, false
	}
	return Step{Kind: "elimination", Technique: name, Explanation: explanation, Details: details, Eliminations: removed, Visual: v}, true
}
func eliminateLiteral(s *Sudoku, lit int) []CandidateElimination {
	c := candidateAt(lit / 2)
	if lit%2 == 0 {
		return []CandidateElimination{{c.Row, c.Col, c.Number}}
	}
	var out []CandidateElimination
	for _, d := range candidateList(s.Candidates[c.Row][c.Col]) {
		if d != c.Number {
			out = append(out, CandidateElimination{c.Row, c.Col, d})
		}
	}
	return out
}
