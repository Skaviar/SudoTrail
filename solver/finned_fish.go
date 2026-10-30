package solver

import "fmt"

// ApplyFinnedXWing retains the existing entry point for a size-2 fish.
func ApplyFinnedXWing(s *Sudoku) (Step, bool) { return ApplyFinnedFish(s, 2) }

// ApplyFinnedFish supports sizes 2–7 in either orientation. Fins are the base
// candidates outside the cover units. A cover candidate outside the base
// units can be removed only when it sees every fin, all in the same box.
func ApplyFinnedFish(s *Sudoku, size int) (Step, bool) {
	if size < 2 || size > 7 {
		return Step{}, false
	}
	for number := 1; number <= 9; number++ {
		for direction := range 2 {
			positions := fishPositions(s, number, direction)
			var eligible uint16
			for base, mask := range positions {
				// A productive fin box needs a covered column/row too, so at
				// most two extra positions can lie outside the N covers.
				if count := candidateCount(mask); count >= 2 && count <= size+2 {
					eligible |= 1 << base
				}
			}
			if candidateCount(eligible) < size {
				continue
			}
			for _, bases := range fishSelections[size] {
				if bases&^eligible != 0 {
					continue
				}
				var allPositions uint16
				for base, mask := range positions {
					if bases&(1<<base) != 0 {
						allPositions |= mask
					}
				}
				if count := candidateCount(allPositions); count <= size || count > size+2 {
					continue // No fins, or too many columns/rows for one fin box.
				}
				for baseGroup := range 3 {
					group := uint16(0b111) << (baseGroup * 3)
					if bases&group == 0 || group&^bases == 0 {
						continue // Need both a fin source and an outside target.
					}
					var outsideGroup uint16
					for base, mask := range positions {
						if bases&(1<<base) != 0 && group&(1<<base) == 0 {
							outsideGroup |= mask
						}
					}
					for positionGroup := range 3 {
						boxPositions := uint16(0b111) << (positionGroup * 3)
						// Every candidate outside this box must be covered.
						// Only its three columns/rows leave any choice, so there
						// are at most eight cover combinations to examine.
						required := outsideGroup | (allPositions &^ boxPositions)
						if candidateCount(required) > size {
							continue
						}
						optional := allPositions & boxPositions &^ required
						box := baseGroup*3 + positionGroup
						if direction == 1 {
							box = positionGroup*3 + baseGroup
						}
						for extra := optional; ; extra = (extra - 1) & optional {
							covers := required | extra
							if candidateCount(covers) == size && covers&boxPositions != 0 {
								if step, ok := finnedFishStep(s, size, number, direction, positions, bases, covers, box); ok {
									return step, true
								}
							}
							if extra == 0 {
								break
							}
						}
					}
				}
			}
		}
	}
	return Step{}, false
}

// The caller has proved that every uncovered base candidate is in box.
// Check the remaining fish and actual targets before building visual evidence.
func finnedFishStep(s *Sudoku, size, number, direction int, positions [9]uint16, bases, covers uint16, box int) (Step, bool) {
	for base, mask := range positions {
		if bases&(1<<base) != 0 && mask&covers == 0 {
			return Step{}, false // Every base must have a location in the core.
		}
	}
	units := allUnits()
	var targets []Cell
	for _, cell := range candidateCells(s, units[18+box], number) {
		base, position := cell.Row, cell.Col
		if direction == 1 {
			base, position = cell.Col, cell.Row
		}
		if bases&(1<<base) == 0 && covers&(1<<position) != 0 && positions[base]&(1<<position) != 0 {
			targets = append(targets, cell)
		}
	}
	if len(targets) == 0 {
		return Step{}, false
	}

	visual := &StepVisual{Digits: []int{number}}
	var fins []Cell
	var baseNames, coverNames []string
	for base := range 9 {
		if bases&(1<<base) == 0 {
			continue
		}
		u := units[direction*9+base]
		visual.Units = append(visual.Units, describeUnit(u))
		baseNames = append(baseNames, u.name)
		cells := candidateCells(s, u, number)
		visual.Sources = append(visual.Sources, cells...)
		if len(cells) == 2 {
			visual.CandidateLinks = append(visual.CandidateLinks, (conjugatePair{cells[0], cells[1], describeUnit(u)}).visualLink(number))
		}
		for _, cell := range cells {
			position := cell.Col
			if direction == 1 {
				position = cell.Row
			}
			if covers&(1<<position) == 0 {
				fins = append(fins, cell)
				visual.Highlights = append(visual.Highlights, CandidateHighlight{cell.Row, cell.Col, number, "fin"})
			}
		}
	}
	for cover := range 9 {
		if covers&(1<<cover) != 0 {
			u := units[(1-direction)*9+cover]
			visual.Units = append(visual.Units, describeUnit(u))
			coverNames = append(coverNames, u.name)
		}
	}
	visual.Units = append(visual.Units, VisualUnit{Kind: "box", Index: box})
	linkEliminations(visual, targets, visual.Sources, number)
	technique := "Finned " + fishNames[size]
	if sashimiCore(positions, bases, covers) {
		technique = "Sashimi " + fishNames[size]
	}
	details := []string{
		fmt.Sprintf("In %s, the possible locations for %d are %s.", formatList(baseNames), number, formatList(cellNames(visual.Sources))),
		fmt.Sprintf("%s cover the pattern except the amber fins at %s.", formatList(coverNames), formatList(cellNames(fins))),
		fmt.Sprintf("If all fins are false, the %d base units must fit %d copies of %d into the %d cover units, excluding other cells in those units.", size, size, number, size),
		fmt.Sprintf("If a fin is %d, every target shares Box %d with all fins, so it cannot be %d.", number, box+1, number),
		"The marked cover cells outside the base units are excluded in either case.",
	}
	if technique == "Sashimi "+fishNames[size] {
		details = append(details, "Without its fins, the core reduces to a single or a smaller fish; this is the Sashimi variant.")
	}
	return applyPatternEliminations(s, technique, fmt.Sprintf("%s on %d, with fins at %s.", technique, number, formatList(cellNames(fins))), number, targets, details, visual)
}

// A fin-free core is Sashimi when a proper subset of bases already fits into
// no more cover units than its size. This includes singles and smaller fish;
// missing intersections alone do not make an ordinary sparse fish Sashimi.
func sashimiCore(positions [9]uint16, bases, covers uint16) bool {
	for subset := (bases - 1) & bases; subset != 0; subset = (subset - 1) & bases {
		var used uint16
		for base, mask := range positions {
			if subset&(1<<base) != 0 {
				used |= mask & covers
			}
		}
		if candidateCount(used) <= candidateCount(subset) {
			return true
		}
	}
	return false
}
