package solver

import "fmt"

type colorComponent struct {
	groups   [2][]Cell
	conflict bool
}

func (c colorComponent) sources() []Cell {
	return append(append([]Cell(nil), c.groups[0]...), c.groups[1]...)
}

// Adjacent nodes must have opposite assignments. An odd cycle contradicts
// that premise, so conflicting components are never used for deductions.
func twoColorComponents(adjacency [81][]int) []colorComponent {
	var visited [81]bool
	var colors [81]int
	var components []colorComponent
	for start := range 81 {
		if visited[start] || len(adjacency[start]) == 0 {
			continue
		}
		visited[start] = true
		queue := []int{start}
		var component colorComponent
		for head := 0; head < len(queue); head++ {
			node := queue[head]
			component.groups[colors[node]] = append(component.groups[colors[node]], Cell{node / 9, node % 9})
			for _, next := range adjacency[node] {
				if !visited[next] {
					visited[next] = true
					colors[next] = 1 - colors[node]
					queue = append(queue, next)
				} else if colors[next] == colors[node] {
					component.conflict = true
				}
			}
		}
		components = append(components, component)
	}
	return components
}

func seesColor(cell Cell, group []Cell) (Cell, bool) {
	for _, source := range group {
		if cellsSeeEachOther(cell, source) {
			return source, true
		}
	}
	return Cell{}, false
}

// ApplySimpleColoring implements Color Wrap and Color Trap using one digit's
// conjugate-pair graph. Colors represent alternate assignments, not guesses.
func ApplySimpleColoring(s *Sudoku) (Step, bool) {
	for number := 1; number <= 9; number++ {
		pairs := conjugatePairs(s, number)
		var adjacency [81][]int
		for _, pair := range pairs {
			a, b := pair.a.Row*9+pair.a.Col, pair.b.Row*9+pair.b.Col
			adjacency[a] = append(adjacency[a], b)
			adjacency[b] = append(adjacency[b], a)
		}
		for _, component := range twoColorComponents(adjacency) {
			if component.conflict {
				continue
			}
			v := &StepVisual{Sources: component.sources(), Digits: []int{number}}
			for color, group := range component.groups {
				for _, cell := range group {
					v.Highlights = append(v.Highlights, CandidateHighlight{cell.Row, cell.Col, number, []string{"color-a", "color-b"}[color]})
				}
			}
			for _, pair := range pairs {
				if containsCell(v.Sources, pair.a) {
					v.CandidateLinks = append(v.CandidateLinks, pair.visualLink(number))
				}
			}
			var wraps [2]bool
			var conflicts [2][2]Cell
			for color, group := range component.groups {
				for i, a := range group {
					for _, b := range group[i+1:] {
						if cellsSeeEachOther(a, b) {
							wraps[color] = true
							conflicts[color] = [2]Cell{a, b}
						}
					}
				}
			}
			if wraps[0] && wraps[1] {
				continue
			}
			for color, bad := range wraps {
				if !bad {
					continue
				}
				a, b := conflicts[color][0], conflicts[color][1]
				v.CandidateLinks = append(v.CandidateLinks, candidateConnection(a, b, number, "weak"))
				details := []string{
					fmt.Sprintf("The only-two-location pairs for %d connect %s; linked cells have opposite colors.", number, formatList(cellNames(v.Sources))),
					"Either all blue candidates are true and all purple candidates false, or vice versa.",
					fmt.Sprintf("Same-colored %s and %s share %s, so that color cannot be true. Remove %d from every candidate of that color.", cellName(a.Row, a.Col), cellName(b.Row, b.Col), unitLabel(sharedUnit(a, b)), number),
				}
				return applyPatternEliminations(s, "Simple Coloring — Color Wrap", fmt.Sprintf("A same-color conflict rules out %d in %s.", number, formatList(cellNames(component.groups[color]))), number, component.groups[color], details, v)
			}
			var targets []Cell
			for row := range 9 {
				for col := range 9 {
					cell := Cell{row, col}
					if !cellHasCandidate(s, cell, number) || containsCell(v.Sources, cell) {
						continue
					}
					a, seesA := seesColor(cell, component.groups[0])
					b, seesB := seesColor(cell, component.groups[1])
					if seesA && seesB {
						targets = append(targets, cell)
						linkEliminations(v, []Cell{cell}, []Cell{a, b}, number)
					}
				}
			}
			if len(targets) == 0 {
				continue
			}
			details := []string{
				fmt.Sprintf("Conjugate pairs for %d alternate between blue and purple candidates in %s.", number, formatList(cellNames(v.Sources))),
				"One whole color must be true. Each target sees at least one candidate of each color.",
				fmt.Sprintf("Whichever color is true, the target cannot contain %d.", number),
			}
			return applyPatternEliminations(s, "Simple Coloring — Color Trap", fmt.Sprintf("Cells seeing both colors cannot contain %d.", number), number, targets, details, v)
		}
	}
	return Step{}, false
}

// ApplyRemotePairs links identical bivalue cells. Adjacent cells must take
// opposite digits; a target seeing both assignments loses either pair digit.
func ApplyRemotePairs(s *Sudoku) (Step, bool) {
	bivalues := bivalueCells(s)
	seenMasks := map[uint16]bool{}
	for _, first := range bivalues {
		mask := s.Candidates[first.Row][first.Col]
		if seenMasks[mask] {
			continue
		}
		seenMasks[mask] = true
		digits := candidateList(mask)
		var cells []Cell
		for _, cell := range bivalues {
			if s.Candidates[cell.Row][cell.Col] == mask {
				cells = append(cells, cell)
			}
		}
		if len(cells) < 4 {
			continue
		}
		var adjacency [81][]int
		for i, a := range cells {
			for _, b := range cells[i+1:] {
				if !cellsSeeEachOther(a, b) {
					continue
				}
				x, y := a.Row*9+a.Col, b.Row*9+b.Col
				adjacency[x] = append(adjacency[x], y)
				adjacency[y] = append(adjacency[y], x)
			}
		}
		for _, component := range twoColorComponents(adjacency) {
			sources := component.sources()
			if component.conflict || len(sources) < 4 {
				continue
			}
			v := &StepVisual{Sources: sources, Digits: digits}
			for color, group := range component.groups {
				for _, cell := range group {
					for i, number := range digits {
						v.Highlights = append(v.Highlights, CandidateHighlight{cell.Row, cell.Col, number, []string{"color-a", "color-b"}[(color+i)%2]})
					}
				}
			}
			for i, a := range sources {
				for _, b := range sources[i+1:] {
					if cellsSeeEachOther(a, b) {
						for _, number := range digits {
							v.CandidateLinks = append(v.CandidateLinks, candidateConnection(a, b, number, "weak"))
						}
					}
				}
			}
			var targets []Cell
			var links [][2]Cell
			for row := range 9 {
				for col := range 9 {
					cell := Cell{row, col}
					if s.Board[row][col] != 0 || s.Candidates[row][col]&mask == 0 || containsCell(sources, cell) {
						continue
					}
					a, seesA := seesColor(cell, component.groups[0])
					b, seesB := seesColor(cell, component.groups[1])
					if seesA && seesB {
						targets = append(targets, cell)
						links = append(links, [2]Cell{a, b})
					}
				}
			}
			if len(targets) == 0 {
				continue
			}
			var eliminations []CandidateElimination
			for i, target := range targets {
				for _, number := range digits {
					if cellHasCandidate(s, target, number) {
						linkEliminations(v, []Cell{target}, links[i][:], number)
					}
				}
			}
			for _, number := range digits {
				eliminations = append(eliminations, removeCandidates(s, targets, number)...)
			}
			return Step{Kind: "elimination", Technique: "Remote Pairs", Visual: v, Eliminations: eliminations,
				Explanation: fmt.Sprintf("A chain of %v pairs rules those digits out of cells seeing opposite assignments.", digits),
				Details: []string{fmt.Sprintf("%s each allow only %v.", formatList(cellNames(sources)), digits),
					"Connected pairs share a row, column, or box, so their values must alternate.",
					fmt.Sprintf("The two color groups must use opposite values: %d and %d, or the reverse.", digits[0], digits[1]),
					fmt.Sprintf("Remove %v from %s, which see both groups.", digits, formatList(cellNames(targets)))}}, true
		}
	}
	return Step{}, false
}
