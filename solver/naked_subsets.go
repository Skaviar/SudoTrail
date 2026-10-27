package solver

import "fmt"

// ApplyNakedSubset reserves N digits for N cells and removes those digits
// from other cells in their shared row, column, or box. Size may be 2, 3, or 4.
func ApplyNakedSubset(s *Sudoku, size int) (Step, bool) {
	if size < 2 || size > 4 {
		return Step{}, false
	}
	technique := map[int]string{2: "Naked Pair", 3: "Naked Triple", 4: "Naked Quadruple"}[size]
	units := allUnits()
	for _, u := range units {
		var eligible []Cell
		for _, cell := range u.cells {
			count := candidateCount(s.Candidates[cell.Row][cell.Col])
			// Singles are handled before subsets. A selected cell cannot have more
			// candidates than the entire subset is allowed to contain.
			if s.Board[cell.Row][cell.Col] == 0 && count >= 2 && count <= size {
				eligible = append(eligible, cell)
			}
		}
		if len(eligible) < size {
			continue
		}
		// Bits here select CELL positions in eligible, not Sudoku digits.
		for selection := uint16(1); selection < uint16(1)<<len(eligible); selection++ {
			if candidateCount(selection) != size {
				continue
			}
			var sources []Cell
			var digits uint16
			for i, cell := range eligible {
				if selection&(1<<i) != 0 {
					sources = append(sources, cell)
					digits |= s.Candidates[cell.Row][cell.Col]
				}
			}
			// The combined candidate set must contain exactly N distinct digits.
			// Cells may have different candidate lists, e.g. {2,5}, {5,8}, {2,8}.
			if candidateCount(digits) != size {
				continue
			}

			shared := sharedSubsetUnits(units, sources)
			var eliminations []CandidateElimination
			var seen [9][9]uint16
			for _, sharedUnit := range shared {
				for _, cell := range sharedUnit.cells {
					if s.Board[cell.Row][cell.Col] != 0 || containsCell(sources, cell) {
						continue
					}
					removable := s.Candidates[cell.Row][cell.Col] & digits &^ seen[cell.Row][cell.Col]
					for _, number := range candidateList(removable) {
						eliminations = append(eliminations, CandidateElimination{Row: cell.Row, Col: cell.Col, Number: number})
					}
					seen[cell.Row][cell.Col] |= removable
				}
			}
			if len(eliminations) == 0 {
				continue
			}

			visual := &StepVisual{Sources: sources, Digits: candidateList(digits)}
			var unitNames []string
			for _, sharedUnit := range shared {
				visual.Units = append(visual.Units, describeUnit(sharedUnit))
				unitNames = append(unitNames, sharedUnit.name)
			}
			details := []string{}
			for _, cell := range sources {
				details = append(details, fmt.Sprintf("%s allows only %v.", cellName(cell.Row, cell.Col), candidateList(s.Candidates[cell.Row][cell.Col])))
			}
			details = append(details,
				fmt.Sprintf("These %d cells must contain digits %s in some order.", size, formatList(maskDigits(digits))),
				fmt.Sprintf("Remove those digits from other cells in %s.", formatList(unitNames)))
			for _, e := range eliminations {
				s.Candidates[e.Row][e.Col] = removeCandidate(s.Candidates[e.Row][e.Col], e.Number)
				details = append(details, fmt.Sprintf("Eliminate %d from %s.", e.Number, cellName(e.Row, e.Col)))
			}
			return Step{
				Kind: "elimination", Technique: technique,
				Explanation: fmt.Sprintf("%s in %s reserves digits %s.", technique, formatList(unitNames), formatList(maskDigits(digits))),
				Details:     details, Eliminations: eliminations, Visual: visual,
			}, true
		}
	}
	return Step{}, false
}

func containsCell(cells []Cell, target Cell) bool {
	for _, cell := range cells {
		if cell == target {
			return true
		}
	}
	return false
}

// A locked pair/triple can share both a box and a line. Apply the deduction
// in every shared unit and report each removed candidate only once.
func sharedSubsetUnits(units []unit, sources []Cell) []unit {
	var shared []unit
	for _, u := range units {
		allInside := true
		for _, cell := range sources {
			if !containsCell(u.cells, cell) {
				allInside = false
				break
			}
		}
		if allInside {
			shared = append(shared, u)
		}
	}
	return shared
}
