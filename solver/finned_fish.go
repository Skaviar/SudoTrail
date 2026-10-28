package solver

import "fmt"

// ApplyFinnedXWing searches two base rows (or columns) and two cover columns
// (or rows). Uncovered candidates are fins; only cover targets seeing every
// fin can be removed. A missing corner makes the pattern Sashimi.
func ApplyFinnedXWing(s *Sudoku) (Step, bool) {
	units := allUnits()
	for number := 1; number <= 9; number++ {
		for direction := range 2 {
			var positions [9]uint16
			for base := range 9 {
				for _, cell := range candidateCells(s, units[direction*9+base], number) {
					position := cell.Col
					if direction == 1 {
						position = cell.Row
					}
					positions[base] |= 1 << position
				}
			}
			for a := 0; a < 8; a++ {
				for b := a + 1; b < 9; b++ {
					if candidateCount(positions[a]) < 2 || candidateCount(positions[a]) > 4 || candidateCount(positions[b]) < 2 || candidateCount(positions[b]) > 4 {
						continue
					}
					for first := 0; first < 8; first++ {
						for second := first + 1; second < 9; second++ {
							covers := uint16(1<<first | 1<<second)
							if (positions[a]|positions[b])&covers != covers || positions[a]&covers == 0 || positions[b]&covers == 0 {
								continue
							}
							var sources, fins []Cell
							for _, base := range []int{a, b} {
								for _, cell := range candidateCells(s, units[direction*9+base], number) {
									sources = append(sources, cell)
									position := cell.Col
									if direction == 1 {
										position = cell.Row
									}
									if covers&(1<<position) == 0 {
										fins = append(fins, cell)
									}
								}
							}
							if len(fins) == 0 {
								continue
							}
							box := boxNumber(fins[0].Row, fins[0].Col)
							sameBox := true
							for _, fin := range fins {
								sameBox = sameBox && boxNumber(fin.Row, fin.Col) == box
							}
							if !sameBox {
								continue
							}
							var targets []Cell
							for _, cover := range []int{first, second} {
								for _, cell := range candidateCells(s, units[(1-direction)*9+cover], number) {
									base := cell.Row
									if direction == 1 {
										base = cell.Col
									}
									if base == a || base == b {
										continue
									}
									seesFins := true
									for _, fin := range fins {
										seesFins = seesFins && cellsSeeEachOther(cell, fin)
									}
									if seesFins {
										targets = append(targets, cell)
									}
								}
							}
							if len(targets) == 0 {
								continue
							}
							technique := "Finned X-Wing"
							if candidateCount(positions[a]&covers) < 2 || candidateCount(positions[b]&covers) < 2 {
								technique = "Sashimi X-Wing"
							}
							v := &StepVisual{Sources: sources, Digits: []int{number}}
							var baseNames, coverNames []string
							for _, base := range []int{a, b} {
								u := units[direction*9+base]
								v.Units = append(v.Units, describeUnit(u))
								baseNames = append(baseNames, u.name)
								cells := candidateCells(s, u, number)
								if len(cells) == 2 {
									v.CandidateLinks = append(v.CandidateLinks, (conjugatePair{cells[0], cells[1], describeUnit(u)}).visualLink(number))
								}
							}
							for _, cover := range []int{first, second} {
								u := units[(1-direction)*9+cover]
								v.Units = append(v.Units, describeUnit(u))
								coverNames = append(coverNames, u.name)
							}
							for _, fin := range fins {
								v.Highlights = append(v.Highlights, CandidateHighlight{fin.Row, fin.Col, number, "fin"})
							}
							linkEliminations(v, targets, sources, number)
							details := []string{
								fmt.Sprintf("In %s, the possible locations for %d are %s.", formatList(baseNames), number, formatList(cellNames(sources))),
								fmt.Sprintf("%s cover the pattern except the fins at %s.", formatList(coverNames), formatList(cellNames(fins))),
								fmt.Sprintf("If all fins are false, the two base units reserve %s for %d.", formatList(coverNames), number),
								fmt.Sprintf("If any fin is %d, any target seeing every fin cannot be %d.", number, number),
								"Remove the digit only from cover cells outside the base units that see all fins.",
							}
							return applyPatternEliminations(s, technique, fmt.Sprintf("%s on %d, with fins at %s.", technique, number, formatList(cellNames(fins))), number, targets, details, v)
						}
					}
				}
			}
		}
	}
	return Step{}, false
}
