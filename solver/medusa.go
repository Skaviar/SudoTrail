package solver

import "fmt"

type candidateColorComponent struct {
	groups [2][]int
	ids    []int
}

func candidateColors(g *candidateGraph, oneDigit bool) []candidateColorComponent {
	var visited [729]bool
	var components []candidateColorComponent
	for _, start := range g.active {
		if visited[start] || len(g.edges[start*2]) == 0 {
			continue
		}
		var color [729]int
		for i := range color {
			color[i] = -1
		}
		color[start] = 0
		visited[start] = true
		component := candidateColorComponent{ids: []int{start}}
		valid := true
		for head := 0; head < len(component.ids); head++ {
			a := component.ids[head]
			component.groups[color[a]] = append(component.groups[color[a]], a)
			for _, edge := range g.edges[a*2] {
				b := edge.to / 2
				if oneDigit && a%9 != b%9 {
					continue
				}
				if color[b] >= 0 {
					valid = valid && color[b] != color[a]
					continue
				}
				color[b] = 1 - color[a]
				visited[b] = true
				component.ids = append(component.ids, b)
			}
		}
		if valid && len(component.ids) > 1 {
			components = append(components, component)
		}
	}
	return components
}

func colorVisual(g *candidateGraph, components ...candidateColorComponent) *StepVisual {
	v := &StepVisual{Exact: true}
	for _, component := range components {
		var included [729]bool
		for color, group := range component.groups {
			for _, id := range group {
				included[id] = true
				c := candidateAt(id)
				if !containsCell(v.Sources, c.cell()) {
					v.Sources = append(v.Sources, c.cell())
				}
				role := "color-a"
				if color == 1 {
					role = "color-b"
				}
				v.Highlights = append(v.Highlights, CandidateHighlight{c.Row, c.Col, c.Number, role})
			}
		}
		for _, id := range component.ids {
			for _, e := range g.edges[id*2] {
				if b := e.to / 2; b > id && included[b] {
					v.Inferences = append(v.Inferences, InferenceLink{candidateAt(id), candidateAt(b), "strong", e.reason})
				}
			}
		}
	}
	return v
}

func conflictsWithColor(id int, group []int) bool {
	for _, other := range group {
		if candidatesConflict(id, other) {
			return true
		}
	}
	return false
}
func colorEliminations(group []int) []CandidateElimination {
	var out []CandidateElimination
	for _, id := range group {
		c := candidateAt(id)
		out = append(out, CandidateElimination{c.Row, c.Col, c.Number})
	}
	return out
}

func ApplyMedusa(s *Sudoku) (Step, bool) { return applyMedusa(s, buildCandidateGraph(s)) }

// Strong links across digits join conjugate pairs through bivalue cells.
// This implements cell/unit conflicts, both-color traps (including cell+unit),
// and a color removing every candidate from a cell or unit/digit constraint.
func applyMedusa(s *Sudoku, g *candidateGraph) (Step, bool) {
	for _, component := range candidateColors(g, false) {
		for color, group := range component.groups {
			conflict := ""
			for _, a := range group {
				for _, b := range group {
					if a < b && candidatesConflict(a, b) {
						conflict = fmt.Sprintf("%s and %s have the same color but cannot both be true.", candidateAt(a), candidateAt(b))
					}
				}
			}
			if conflict == "" {
				for _, constraint := range g.constraints {
					empty := true
					for _, id := range constraint {
						empty = empty && conflictsWithColor(id, group)
					}
					if empty {
						conflict = "This color would remove every option from a cell or every location of a digit in a unit."
						break
					}
				}
			}
			if conflict != "" {
				return commitEliminations(s, "3D Medusa", "One entire color is impossible.", colorEliminations(group), []string{conflict, fmt.Sprintf("Remove all candidates of color %d; the opposite color must be true.", color+1)}, colorVisual(g, component))
			}
		}
		var targets []CandidateElimination
		for _, id := range g.active {
			if conflictsWithColor(id, component.groups[0]) && conflictsWithColor(id, component.groups[1]) {
				c := candidateAt(id)
				targets = append(targets, CandidateElimination{c.Row, c.Col, c.Number})
			}
		}
		if len(targets) > 0 {
			return commitEliminations(s, "3D Medusa", "These candidates conflict with both possible color assignments.", targets, []string{"Every strong link has opposite colors at its ends.", "All candidates of one color must be true. Each removed candidate conflicts with a candidate of each color, through a unit or its own cell."}, colorVisual(g, component))
		}
	}
	return Step{}, false
}

func ApplyMultiColoring(s *Sudoku) (Step, bool) { return applyMultiColoring(s, buildCandidateGraph(s)) }

func applyMultiColoring(s *Sudoku, g *candidateGraph) (Step, bool) {
	components := candidateColors(g, true)
	for i, a := range components {
		for _, b := range components[i+1:] {
			if a.ids[0]%9 != b.ids[0]%9 {
				continue
			}
			for ac := range 2 {
				for bc := range 2 {
					var left, right int
					linked := false
					for _, x := range a.groups[ac] {
						for _, y := range b.groups[bc] {
							if candidatesConflict(x, y) {
								left, right, linked = x, y, true
							}
						}
					}
					if !linked {
						continue
					}
					v := colorVisual(g, a, b)
					v.Inferences = append(v.Inferences, InferenceLink{candidateAt(left), candidateAt(right), "weak", "Different color groups cannot both be true here."})
					// If A is true, B's opposite color is true. If A is false,
					// A's opposite color is true. Eliminate anything seeing both.
					var targets []CandidateElimination
					for _, id := range g.active {
						if id%9 == left%9 && conflictsWithColor(id, a.groups[1-ac]) && conflictsWithColor(id, b.groups[1-bc]) {
							c := candidateAt(id)
							targets = append(targets, CandidateElimination{c.Row, c.Col, c.Number})
						}
					}
					if len(targets) > 0 {
						return commitEliminations(s, "Multi-Coloring", "A weak link joins two separate coloring components.", targets, []string{"The linked colors cannot both be true, so at least one opposite color must be true.", "Each removed candidate sees both opposite colors."}, v)
					}
					// If one color conflicts with both colors in the other
					// component, the entire first color is impossible (type 2).
					if conflictsBetweenColors(a.groups[ac], b.groups[1-bc]) {
						return commitEliminations(s, "Multi-Coloring", "This color conflicts with both assignments of another component.", colorEliminations(a.groups[ac]), []string{"At least one color of the second component is true; either choice rules out the indicated color of the first component."}, v)
					}
				}
			}
		}
	}
	return Step{}, false
}
func conflictsBetweenColors(a, b []int) bool {
	for _, id := range a {
		if conflictsWithColor(id, b) {
			return true
		}
	}
	return false
}
