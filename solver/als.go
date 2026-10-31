package solver

import "fmt"

// An ALS has N cells in one house and N+1 digits. If one digit is absent,
// all the other digits must be used. That pigeonhole argument is its link.
type almostLockedSet struct {
	cells     cellSet
	digits    uint16
	positions [10]cellSet
}
type alsEdge struct{ to, digit int }
type alsGraph struct {
	sets  []almostLockedSet
	edges [][]alsEdge
}

func buildALS(s *Sudoku) *alsGraph {
	g := &alsGraph{}
	seen := map[cellSet]bool{}
	for _, u := range allUnits() {
		var cells []Cell
		for _, c := range u.cells {
			if s.Board[c.Row][c.Col] == 0 && candidateCount(s.Candidates[c.Row][c.Col]) >= 2 {
				cells = append(cells, c)
			}
		}
		for selection := 1; selection < 1<<len(cells); selection++ {
			var a almostLockedSet
			for j, c := range cells {
				if selection&(1<<j) == 0 {
					continue
				}
				i := indexCell(c)
				a.cells.add(i)
				mask := s.Candidates[c.Row][c.Col]
				a.digits |= mask
				for _, d := range candidateList(mask) {
					a.positions[d].add(i)
				}
			}
			if candidateCount(a.digits) == a.cells.count()+1 && !seen[a.cells] {
				seen[a.cells] = true
				g.sets = append(g.sets, a)
			}
		}
	}
	g.edges = make([][]alsEdge, len(g.sets))
	for i, a := range g.sets {
		for j := i + 1; j < len(g.sets); j++ {
			b := g.sets[j]
			for _, d := range candidateList(a.digits & b.digits) {
				// No RCC may occur in an overlapping cell. Every occurrence
				// on either side must see EVERY occurrence on the other side.
				if b.positions[d].minus(commonPeers(a.positions[d])).empty() {
					g.edges[i] = append(g.edges[i], alsEdge{j, d})
					g.edges[j] = append(g.edges[j], alsEdge{i, d})
				}
			}
		}
	}
	return g
}

func alsVisual(g *alsGraph, path []int, digits []int) (*StepVisual, []string) {
	v := &StepVisual{Exact: true}
	var details []string
	for j, index := range path {
		a := g.sets[index]
		for _, c := range a.cells.cells() {
			if !containsCell(v.Sources, c) {
				v.Sources = append(v.Sources, c)
			}
		}
		details = append(details, fmt.Sprintf("ALS %d: %s, digits %v (%d cells, %d digits).", j+1, formatList(cellNames(a.cells.cells())), candidateList(a.digits), a.cells.count(), candidateCount(a.digits)))
		if j == 0 {
			continue
		}
		d := digits[j-1]
		previous := g.sets[path[j-1]]
		for _, a := range previous.positions[d].cells() {
			for _, b := range g.sets[index].positions[d].cells() {
				v.Inferences = append(v.Inferences, InferenceLink{Candidate{a.Row, a.Col, d}, Candidate{b.Row, b.Col, d}, "weak", "Restricted common candidate: at most one ALS can contain this digit."})
			}
		}
		details = append(details, fmt.Sprintf("ALS %d and %d are restricted on %d: all its occurrences see one another.", j, j+1, d))
	}
	return v, details
}

func ApplyALS(s *Sudoku) (Step, bool) { return applyALS(s, buildALS(s)) }

func applyALS(s *Sudoku, g *alsGraph) (Step, bool) {
	// Doubly linked sets: both sets must use every non-RCC digit, and
	// each RCC must be in one of the sets. Check this before ordinary chains.
	for i, a := range g.sets {
		for _, e := range g.edges[i] {
			if e.to <= i {
				continue
			}
			var rcc uint16
			for _, other := range g.edges[i] {
				if other.to == e.to {
					rcc |= 1 << other.digit
				}
			}
			if candidateCount(rcc) != 2 {
				continue
			}
			b := g.sets[e.to]
			var targets []CandidateElimination
			for d := 1; d <= 9; d++ {
				var positions []cellSet
				if hasCandidate(rcc, d) {
					positions = append(positions, a.positions[d].or(b.positions[d]))
				} else {
					if hasCandidate(a.digits, d) {
						positions = append(positions, a.positions[d])
					}
					if hasCandidate(b.digits, d) {
						positions = append(positions, b.positions[d])
					}
				}
				for _, p := range positions {
					for _, c := range digitCells(s, d).and(commonPeers(p)).cells() {
						targets = append(targets, CandidateElimination{c.Row, c.Col, d})
					}
				}
			}
			if len(targets) > 0 {
				v, details := alsVisual(g, []int{i, e.to}, []int{e.digit})
				details = append(details, fmt.Sprintf("There are two RCCs, %v. They must be split between the ALSs; each non-RCC digit is locked in its own ALS.", candidateList(rcc)))
				return commitEliminations(s, "Doubly Linked ALS-XZ", "Two restricted common candidates lock the other digits into their ALSs.", targets, details, v)
			}
		}
	}
	// A state means "digit is absent from this ALS". Each edge propagates
	// that absence through an RCC; no cell values are guessed or assigned.
	for start, a := range g.sets {
		for _, z := range candidateList(a.digits) {
			targetsAtStart := digitCells(s, z).and(commonPeers(a.positions[z]))
			if targetsAtStart.empty() {
				continue
			}
			type state struct{ index, missing, parent, via int }
			queue := []state{{start, z, -1, 0}}
			seen := map[[2]int]bool{{start, z}: true}
			for head := 0; head < len(queue); head++ {
				current := queue[head]
				end := g.sets[current.index]
				if head > 0 && current.missing != z && hasCandidate(end.digits, z) {
					possible := targetsAtStart.and(commonPeers(end.positions[z]))
					if !possible.empty() {
						var reverse []int
						var occupied cellSet
						for j := head; j >= 0; j = queue[j].parent {
							reverse = append(reverse, j)
							occupied = occupied.or(g.sets[queue[j].index].cells)
						}
						possible = possible.minus(occupied)
						if !possible.empty() {
							var path, links []int
							for j := len(reverse) - 1; j >= 0; j-- {
								q := queue[reverse[j]]
								path = append(path, q.index)
								if j < len(reverse)-1 {
									links = append(links, q.via)
								}
							}
							name := "ALS Chain"
							if len(path) == 2 {
								name = "ALS-XZ"
							}
							if len(path) == 3 {
								name = "ALS-XY-Wing"
							}
							v, details := alsVisual(g, path, links)
							details = append(details, fmt.Sprintf("If %d is absent from the first ALS, successive RCCs force it into the last ALS. At least one endpoint ALS contains %d.", z, z))
							return applyPatternEliminations(s, name, "At least one endpoint ALS must contain the eliminated digit.", z, possible.cells(), details, v)
						}
					}
				}
				for _, e := range g.edges[current.index] {
					if e.digit == current.missing || seen[[2]int{e.to, e.digit}] {
						continue
					}
					// Distinct ALS nodes keep the proof easy to follow and
					// avoid recycling an ALS through different missing digits.
					repeated := false
					for j := head; j >= 0; j = queue[j].parent {
						repeated = repeated || queue[j].index == e.to
					}
					if repeated {
						continue
					}
					seen[[2]int{e.to, e.digit}] = true
					queue = append(queue, state{e.to, e.digit, head, e.digit})
				}
			}
		}
	}
	return Step{}, false
}

func ApplyDeathBlossom(s *Sudoku) (Step, bool) { return applyDeathBlossom(s, buildALS(s)) }

func applyDeathBlossom(s *Sudoku, g *alsGraph) (Step, bool) {
	for stem := range 81 {
		c := cellIndex(stem)
		mask := s.Candidates[c.Row][c.Col]
		if s.Board[c.Row][c.Col] != 0 || candidateCount(mask) < 2 {
			continue
		}
		for z := 1; z <= 9; z++ {
			if hasCandidate(mask, z) {
				continue
			}
			for _, target := range digitCells(s, z).cells() {
				var petals, links []int
				for _, d := range candidateList(mask) {
					found := -1
					for j, a := range g.sets {
						if a.cells.has(stem) || a.cells.has(indexCell(target)) || !hasCandidate(a.digits, z) || !hasCandidate(a.digits, d) {
							continue
						}
						if a.positions[d].minus(topology.peers[stem]).empty() && a.positions[z].minus(topology.peers[indexCell(target)]).empty() {
							found = j
							break
						}
					}
					if found < 0 {
						break
					}
					petals = append(petals, found)
					links = append(links, d)
				}
				if len(petals) != candidateCount(mask) {
					continue
				}
				v := &StepVisual{Sources: []Cell{c}, Exact: true}
				var details []string
				for j, index := range petals {
					a := g.sets[index]
					d := links[j]
					details = append(details, fmt.Sprintf("If %s=%d, the ALS at %s loses %d and must contain %d.", cellName(c.Row, c.Col), d, formatList(cellNames(a.cells.cells())), d, z))
					for _, p := range a.cells.cells() {
						if !containsCell(v.Sources, p) {
							v.Sources = append(v.Sources, p)
						}
					}
					for _, p := range a.positions[d].cells() {
						v.Inferences = append(v.Inferences, InferenceLink{Candidate{c.Row, c.Col, d}, Candidate{p.Row, p.Col, d}, "weak", "The stem value removes this RCC from the petal."})
					}
				}
				return applyPatternEliminations(s, "Death Blossom", "Every possible stem value locks the same digit into a petal seen by the target.", z, []Cell{target}, details, v)
			}
		}
	}
	return Step{}, false
}
