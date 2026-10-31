package solver

import "fmt"

type implicationResult struct {
	value [729]int8 // 0 unknown; 1 false; 2 true.
	node  [1458]int
	proof ProofBranch
}

// propagateAssumption performs deterministic constraint propagation under ONE
// premise. A net can combine earlier deductions to create a new single. There
// is no recursive branching, guessed completion, or call to countSolutions.
func propagateAssumption(g *candidateGraph, root int, net bool) implicationResult {
	r := implicationResult{}
	for i := range r.node {
		r.node[i] = -1
	}
	var queue []int
	assign := func(lit int, parents []int, reason string) {
		if r.node[lit] >= 0 || r.proof.Contradiction != "" {
			return
		}
		index := len(r.proof.Nodes)
		r.node[lit] = index
		r.proof.Nodes = append(r.proof.Nodes, ProofNode{literalAt(lit), parents, reason})
		if other := r.node[lit^1]; other >= 0 {
			r.proof.Contradiction = fmt.Sprintf("Both %s and its negation follow.", candidateAt(lit/2))
			r.proof.Conflict = []int{other, index}
			return
		}
		r.value[lit/2] = int8(lit%2 + 1)
		queue = append(queue, lit)
	}
	assign(root, nil, "Assumption")
	for head := 0; head < len(queue) && r.proof.Contradiction == ""; head++ {
		lit := queue[head]
		for _, e := range g.edges[lit] {
			assign(e.to, []int{r.node[lit]}, e.reason)
		}
		if lit%2 == 1 {
			continue
		}
		for _, u := range g.memberships[lit/2] {
			remaining, last := 0, 0
			var parents []int
			for _, id := range g.constraints[u] {
				if r.value[id] == 1 {
					parents = append(parents, r.node[id*2])
				} else {
					remaining++
					last = id
				}
			}
			if remaining == 0 && r.proof.Contradiction == "" {
				r.proof.Contradiction = "No possible candidate remains for " + g.constraintNames[u] + "."
				r.proof.Conflict = parents
			}
			if net && remaining == 1 {
				assign(last*2+1, parents, "Only option left for "+g.constraintNames[u])
			}
		}
	}
	return r
}

// Keep only the part of a branch needed for this conclusion or contradiction.
// Parents retain their earlier-node indices after compacting the proof DAG.
func compactProof(proof ProofBranch, ends []int) ProofBranch {
	if proof.Contradiction != "" {
		ends = append(ends, proof.Conflict...)
	}
	needed := make([]bool, len(proof.Nodes))
	var visit func(int)
	visit = func(i int) {
		if i < 0 || i >= len(needed) || needed[i] {
			return
		}
		needed[i] = true
		for _, p := range proof.Nodes[i].Parents {
			visit(p)
		}
	}
	for _, end := range ends {
		visit(end)
	}
	mapping := make([]int, len(needed))
	out := ProofBranch{Contradiction: proof.Contradiction}
	for i, node := range proof.Nodes {
		if !needed[i] {
			continue
		}
		mapping[i] = len(out.Nodes)
		node.Parents = append([]int(nil), node.Parents...)
		for j, p := range node.Parents {
			node.Parents[j] = mapping[p]
		}
		out.Nodes = append(out.Nodes, node)
	}
	for _, i := range proof.Conflict {
		out.Conflict = append(out.Conflict, mapping[i])
	}
	return out
}

func proofVisual(branches []ProofBranch) (*StepVisual, []string) {
	v := &StepVisual{Exact: true, Proofs: branches}
	var details []string
	for b, proof := range branches {
		details = append(details, fmt.Sprintf("Branch %d:", b+1))
		for i, node := range proof.Nodes {
			lit := candidateID(node.Row*9+node.Col, node.Number) * 2
			if node.True {
				lit++
			}
			details = append(details, fmt.Sprintf("%d. %s (%s).", i+1, literalText(lit), node.Reason))
			if !containsCell(v.Sources, node.Candidate.cell()) {
				v.Sources = append(v.Sources, node.Candidate.cell())
			}
			for _, p := range node.Parents {
				parent := proof.Nodes[p]
				// Dynamic singles have several premises. Their edges are
				// implications in a net, not invented pairwise strong links.
				v.Inferences = append(v.Inferences, InferenceLink{parent.Candidate, node.Candidate, "implication", node.Reason})
			}
		}
		if proof.Contradiction != "" {
			details = append(details, proof.Contradiction)
		}
	}
	return v, details
}

func ApplyForcingChains(s *Sudoku) (Step, bool) {
	return applyForcing(s, buildCandidateGraph(s), false)
}
func ApplyForcingNets(s *Sudoku) (Step, bool) { return applyForcing(s, buildCandidateGraph(s), true) }

func applyForcing(s *Sudoku, g *candidateGraph, net bool) (Step, bool) {
	name := "Forcing Chain"
	if net {
		name = "Forcing Net"
	}
	// Cache the deterministic result of a premise within this deduction only.
	// Nothing survives a candidate change or a different benchmark worker.
	cache := make(map[int]*implicationResult)
	get := func(root int) *implicationResult {
		if r := cache[root]; r != nil {
			return r
		}
		r := propagateAssumption(g, root, net)
		cache[root] = &r
		return &r
	}
	for _, id := range g.active {
		for _, truth := range []int{1, 0} {
			root := id*2 + truth
			r := get(root)
			if r.proof.Contradiction != "" {
				if len(eliminateLiteral(s, root^1)) == 0 {
					continue
				}
				v, details := proofVisual([]ProofBranch{compactProof(r.proof, nil)})
				return commitEliminations(s, name, fmt.Sprintf("The assumption leads to a contradiction; %s.", literalText(root^1)), eliminateLiteral(s, root^1), details, v)
			}
		}
		a, b := get(id*2), get(id*2+1)
		for _, target := range g.active {
			if target == id || a.value[target] == 0 || a.value[target] != b.value[target] {
				continue
			}
			lit := target*2 + int(a.value[target]) - 1
			if len(eliminateLiteral(s, lit)) == 0 {
				continue
			}
			v, details := proofVisual([]ProofBranch{compactProof(a.proof, []int{a.node[lit]}), compactProof(b.proof, []int{b.node[lit]})})
			return commitEliminations(s, name, fmt.Sprintf("Both possibilities prove %s.", literalText(lit)), eliminateLiteral(s, lit), details, v)
		}
	}
	// Multiple forcing: every possible value of a cell, or every possible
	// location of a digit in a house, leads to the same conclusion.
	for _, constraint := range g.constraints {
		if len(constraint) < 3 {
			continue
		}
		var branches []*implicationResult
		for _, id := range constraint {
			branches = append(branches, get(id*2+1))
		}
		for _, target := range g.active {
			value := branches[0].value[target]
			if value == 0 {
				continue
			}
			common := true
			for _, b := range branches[1:] {
				common = common && b.value[target] == value
			}
			if !common {
				continue
			}
			lit := target*2 + int(value) - 1
			if len(eliminateLiteral(s, lit)) == 0 {
				continue
			}
			var proofs []ProofBranch
			for _, b := range branches {
				proofs = append(proofs, compactProof(b.proof, []int{b.node[lit]}))
			}
			v, details := proofVisual(proofs)
			return commitEliminations(s, name, fmt.Sprintf("Every option in this cell or unit proves %s.", literalText(lit)), eliminateLiteral(s, lit), details, v)
		}
	}
	return Step{}, false
}
