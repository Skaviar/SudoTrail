package solver

import "fmt"

func ApplyXChains(s *Sudoku) (Step, bool)  { return applyChains(s, buildCandidateGraph(s), "X-Chain") }
func ApplyXYChains(s *Sudoku) (Step, bool) { return applyChains(s, buildCandidateGraph(s), "XY-Chain") }
func ApplyAIC(s *Sudoku) (Step, bool) {
	return applyChains(s, buildCandidateGraph(s), "Alternating Inference Chain")
}

// Reachability follows alternating true->false weak links and false->true
// strong links. It never fills a trial board or chooses a second assumption.
// A path from a proposition to its negation is a discontinuous Nice Loop.
// Ordinary endpoint eliminations and continuous-loop eliminations have the
// same proof when the eliminated candidate is used as the starting premise.
func chainReach(g *candidateGraph, root int, mode string) (parent [1458]int, reasons [1458]string, order []int) {
	for i := range parent {
		parent[i] = -1
	}
	parent[root] = root
	order = append(order, root)
	for head := 0; head < len(order); head++ {
		from := order[head]
		for _, e := range g.edges[from] {
			if mode == "X-Chain" && from/2%9 != e.to/2%9 {
				continue
			}
			if mode == "XY-Chain" && (from%2 == 0 && !e.cell || from%2 == 1 && e.cell) {
				continue
			}
			if parent[e.to] >= 0 {
				continue
			}
			parent[e.to] = from
			reasons[e.to] = e.reason
			order = append(order, e.to)
		}
	}
	return
}

func chainProof(v *StepVisual, parent [1458]int, reasons [1458]string, end int) []string {
	var path []int
	for {
		path = append(path, end)
		if parent[end] == end {
			break
		}
		end = parent[end]
	}
	var details []string
	for i := len(path) - 1; i > 0; i-- {
		a, b := path[i], path[i-1]
		addInference(v, a, b, reasons[b])
		details = append(details, fmt.Sprintf("If %s, then %s (%s).", literalText(a), literalText(b), reasons[b]))
	}
	return details
}

func applyChains(s *Sudoku, g *candidateGraph, mode string) (Step, bool) {
	for _, id := range g.active {
		for _, truth := range []int{1, 0} {
			root := id*2 + truth
			parent, reasons, _ := chainReach(g, root, mode)
			if parent[root^1] < 0 {
				continue
			}
			if len(eliminateLiteral(s, root^1)) == 0 {
				continue
			}
			v := &StepVisual{Exact: true}
			details := chainProof(v, parent, reasons, root^1)
			details = append(details, fmt.Sprintf("The assumption contradicts itself, so %s.", literalText(root^1)))
			return commitEliminations(s, mode, fmt.Sprintf("Following alternating links proves %s.", literalText(root^1)), eliminateLiteral(s, root^1), details, v)
		}
	}
	return Step{}, false
}
