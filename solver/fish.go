package solver

import "fmt"

var fishNames = [...]string{2: "X-Wing", 3: "Swordfish", 4: "Jellyfish", 5: "Squirmbag", 6: "Whale", 7: "Leviathan"}

// These masks select zero-based rows/columns, not Sudoku digits. The fixed
// combinations are shared across puzzles and are never modified by a solver.
var fishSelections = func() [8][]uint16 {
	var selections [8][]uint16
	for mask := uint16(1); mask < 1<<9; mask++ {
		if size := candidateCount(mask); size >= 2 && size <= 7 {
			selections[size] = append(selections[size], mask)
		}
	}
	return selections
}()

// In row mode, each mask holds a row's candidate columns. Column mode swaps
// the axes. A unit already containing the digit has no candidate locations.
func fishPositions(s *Sudoku, number, direction int) [9]uint16 {
	var positions [9]uint16
	for base := range 9 {
		for position := range 9 {
			cell := fishCell(base, position, direction)
			if int(s.Board[cell.Row][cell.Col]) == number {
				positions[base] = 0
				break
			}
			if cellHasCandidate(s, cell, number) {
				positions[base] |= 1 << position
			}
		}
	}
	return positions
}

func fishCell(base, position, direction int) Cell {
	if direction == 1 {
		return Cell{position, base}
	}
	return Cell{base, position}
}

// ApplyBasicFish reserves N columns for N rows, or N rows for N columns,
// considering one digit at a time. Sizes 2, 3, and 4 are X-Wing, Swordfish,
// and Jellyfish; sizes 5–7 are Squirmbag, Whale, and Leviathan.
// Intersections need not all contain the digit.
func ApplyBasicFish(s *Sudoku, size int) (Step, bool) {
	if size < 2 || size > 7 {
		return Step{}, false
	}
	technique := fishNames[size]
	units := allUnits()
	for number := 1; number <= 9; number++ {
		for direction := range 2 {
			positions := fishPositions(s, number, direction)
			var eligible uint16
			for base := range 9 {
				count := candidateCount(positions[base])
				if count >= 2 && count <= size {
					eligible |= 1 << base
				}
			}
			if candidateCount(eligible) < size {
				continue
			}
			for _, selection := range fishSelections[size] {
				if selection&^eligible != 0 {
					continue
				}
				var covers uint16
				for base := range 9 {
					if selection&(1<<base) != 0 {
						covers |= positions[base]
					}
				}
				if candidateCount(covers) != size {
					continue
				}
				visual := &StepVisual{Digits: []int{number}}
				var baseNames, coverNames, details []string
				for base := range 9 {
					if selection&(1<<base) == 0 {
						continue
					}
					u := units[direction*9+base]
					cells := candidateCells(s, u, number)
					visual.Sources = append(visual.Sources, cells...)
					visual.Units = append(visual.Units, describeUnit(u))
					baseNames = append(baseNames, u.name)
					details = append(details, fmt.Sprintf("%s can place %d only in %s.", u.name, number, formatList(cellNames(cells))))
					if len(cells) == 2 {
						pair := conjugatePair{cells[0], cells[1], describeUnit(u)}
						visual.CandidateLinks = append(visual.CandidateLinks, pair.visualLink(number))
					}
				}
				var targets []Cell
				for cover := range 9 {
					if covers&(1<<cover) == 0 {
						continue
					}
					u := units[(1-direction)*9+cover]
					visual.Units = append(visual.Units, describeUnit(u))
					coverNames = append(coverNames, u.name)
					for _, cell := range candidateCells(s, u, number) {
						base := cell.Row
						if direction == 1 {
							base = cell.Col
						}
						if selection&(1<<base) == 0 {
							targets = append(targets, cell)
						}
					}
				}
				if len(targets) == 0 {
					continue
				}
				details = append(details, fmt.Sprintf("These %d units must place %d in different cells of %s, reserving those units for this pattern.", size, number, formatList(coverNames)))
				linkEliminations(visual, targets, visual.Sources, number)
				return applyPatternEliminations(s, technique,
					fmt.Sprintf("%s on %d: %s reserve %s.", technique, number, formatList(baseNames), formatList(coverNames)),
					number, targets, details, visual)
			}
		}
	}
	return Step{}, false
}
