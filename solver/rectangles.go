package solver

import "fmt"

var rectangles = func() (out [][4]Cell) {
	for r := 0; r < 8; r++ {
		for rr := r + 1; rr < 9; rr++ {
			for c := 0; c < 8; c++ {
				for cc := c + 1; cc < 9; cc++ {
					if (r/3 == rr/3) != (c/3 == cc/3) {
						out = append(out, [4]Cell{{r, c}, {r, cc}, {rr, c}, {rr, cc}})
					}
				}
			}
		}
	}
	return
}()

// Missing-candidate rectangles are checked against the ORIGINAL clues. A digit
// removed by an earlier logical step may be restored for pattern recognition;
// a digit blocked by a given must never be restored, even for recognition.
func ApplyAdvancedRectangles(s *Sudoku) (Step, bool)         { return applyRectangles(s, false, "") }
func ApplyMissingCandidateRectangles(s *Sudoku) (Step, bool) { return applyRectangles(s, true, "") }
func ApplyHiddenRectangle(s *Sudoku) (Step, bool) {
	return applyRectangles(s, false, "Hidden Rectangle")
}

func rectanglePairAllowed(cells []Cell, allowed [9][9]uint16, pair uint16) bool {
	for _, c := range cells {
		if allowed[c.Row][c.Col]&pair != pair {
			return false
		}
	}
	return true
}

func applyRectangles(s *Sudoku, missing bool, requested string) (Step, bool) {
	var allowed [9][9]uint16
	if missing {
		if !s.initialKnown {
			return Step{}, false
		}
		original := &Sudoku{Board: s.initialBoard}
		original.initializeCandidates()
		allowed = original.Candidates
	}
	for _, rectangle := range rectangles {
		cells := rectangle[:]
		unsolved := true
		for _, c := range cells {
			unsolved = unsolved && s.Board[c.Row][c.Col] == 0
		}
		if !unsolved {
			continue
		}
		for a := 1; a < 9; a++ {
			for b := a + 1; b <= 9; b++ {
				pair := uint16(1<<a | 1<<b)
				if missing && !rectanglePairAllowed(cells, allowed, pair) {
					continue
				}
				valid, hasMissing := true, false
				var roofs []Cell
				for _, c := range cells {
					mask := s.Candidates[c.Row][c.Col]
					if mask&pair != pair {
						hasMissing = true
						if !missing || allowed[c.Row][c.Col]&pair != pair || mask&pair == 0 {
							valid = false
							break
						}
					}
					if mask&^pair != 0 {
						roofs = append(roofs, c)
					}
				}
				if !valid || missing != hasMissing || len(roofs) == 0 || len(roofs) == 4 {
					continue
				}
				v := &StepVisual{Sources: append([]Cell(nil), cells...), Digits: []int{a, b}, Exact: true}
				var targets []CandidateElimination
				name := ""
				details := []string{fmt.Sprintf("%s form a rectangle on %d/%d in exactly two boxes.", formatList(cellNames(cells)), a, b), "The original clues have exactly one solution. At least one extra candidate must prevent a swappable rectangle."}
				add := func(cs []Cell, d int) {
					for _, c := range cs {
						if cellHasCandidate(s, c, d) {
							targets = append(targets, CandidateElimination{c.Row, c.Col, d})
						}
					}
				}
				adjacent := len(roofs) == 2 && (roofs[0].Row == roofs[1].Row || roofs[0].Col == roofs[1].Col)
				if missing && len(roofs) == 1 {
					name = "Unique Rectangle Type 1"
					add(roofs, a)
					add(roofs, b)
				}
				if len(roofs) >= 2 {
					var extras uint16
					for _, c := range roofs {
						extras |= s.Candidates[c.Row][c.Col] &^ pair
					}
					if candidateCount(extras) == 1 && (len(roofs) == 3 || !adjacent || missing) {
						d := singleCandidate(extras)
						name = "Unique Rectangle Type 5"
						if adjacent {
							name = "Unique Rectangle Type 2"
						}
						add(commonCandidatePeers(s, roofs, d), d)
						details = append(details, fmt.Sprintf("One of the roof cells must contain the extra digit %d.", d))
					}
				}
				if len(targets) == 0 && adjacent {
					// Type 4: a rectangle digit is conjugate in the two roofs.
					for _, u := range allUnits() {
						if !containsCell(u.cells, roofs[0]) || !containsCell(u.cells, roofs[1]) {
							continue
						}
						for _, d := range []int{a, b} {
							positions := candidateCells(s, u, d)
							if len(positions) == 2 && containsCell(positions, roofs[0]) && containsCell(positions, roofs[1]) {
								name = "Unique Rectangle Type 4"
								other := a
								if d == a {
									other = b
								}
								add(roofs, other)
								details = append(details, fmt.Sprintf("%d must be in one roof in %s; neither roof may use %d as well, which would create the deadly rectangle.", d, u.name, other))
								break
							}
						}
						if len(targets) > 0 {
							break
						}
					}
					// Type 3: treat the union of roof extras as one virtual cell.
					if len(targets) == 0 {
						for _, u := range allUnits() {
							if !containsCell(u.cells, roofs[0]) || !containsCell(u.cells, roofs[1]) {
								continue
							}
							var other []Cell
							for _, c := range u.cells {
								if !containsCell(cells, c) && s.Board[c.Row][c.Col] == 0 {
									other = append(other, c)
								}
							}
							for subset := 1; subset < 1<<len(other); subset++ {
								selected := []Cell{roofs[0], roofs[1]}
								digits := (s.Candidates[roofs[0].Row][roofs[0].Col] | s.Candidates[roofs[1].Row][roofs[1].Col]) &^ pair
								for i, c := range other {
									if subset&(1<<i) != 0 {
										selected = append(selected, c)
										digits |= s.Candidates[c.Row][c.Col]
									}
								}
								if candidateCount(digits) != len(selected)-1 {
									continue
								}
								for _, c := range u.cells {
									if !containsCell(selected, c) && !containsCell(cells, c) {
										for _, d := range candidateList(digits) {
											add([]Cell{c}, d)
										}
									}
								}
								if len(targets) > 0 {
									name = "Unique Rectangle Type 3"
									v.Sources = append(v.Sources, selected[2:]...)
									details = append(details, fmt.Sprintf("The roof extras act as one virtual cell. Together with %s they lock digits %v into %s.", formatList(cellNames(selected[2:])), candidateList(digits), u.name))
									break
								}
							}
							if len(targets) > 0 {
								break
							}
						}
					}
				}
				if len(targets) == 0 && len(roofs) == 2 && !adjacent {
					// Type 6: a conjugate X-Wing on a rectangle digit.
					for _, d := range []int{a, b} {
						locked := true
						for _, u := range []int{cells[0].Row, cells[2].Row, 9 + cells[0].Col, 9 + cells[1].Col} {
							positions := candidateCells(s, allUnits()[u], d)
							if len(positions) != 2 {
								locked = false
								break
							}
							for _, c := range positions {
								locked = locked && containsCell(cells, c)
							}
						}
						if locked {
							name = "Unique Rectangle Type 6"
							add(roofs, d)
							details = append(details, fmt.Sprintf("The %ds form an X-Wing; putting %d in either diagonal roof would force the deadly rectangle.", d, d))
							break
						}
					}
				}
				if len(targets) == 0 {
					// Hidden rectangle: start at a floor; two conjugate links
					// meeting at its opposite corner force the deadly pattern
					// if the other rectangle digit is placed at that corner.
					for i, floor := range cells {
						if s.Candidates[floor.Row][floor.Col]&^pair != 0 {
							continue
						}
						opposite := cells[3-i]
						for _, d := range []int{a, b} {
							locked := true
							for _, u := range []int{opposite.Row, 9 + opposite.Col} {
								positions := candidateCells(s, allUnits()[u], d)
								if len(positions) != 2 {
									locked = false
									break
								}
								for _, c := range positions {
									locked = locked && containsCell(cells, c)
								}
							}
							if locked {
								other := a
								if d == a {
									other = b
								}
								name = "Hidden Rectangle"
								add([]Cell{opposite}, other)
								details = append(details, fmt.Sprintf("Conjugate links for %d meet at %s, opposite the floor %s.", d, cellName(opposite.Row, opposite.Col), cellName(floor.Row, floor.Col)))
								break
							}
						}
						if len(targets) > 0 {
							break
						}
					}
				}
				if len(targets) > 0 {
					if requested != "" && name != requested {
						continue
					}
					if !s.HasUniqueSolution() {
						return Step{}, false
					}
					if missing {
						name += " (Missing Candidates)"
						details = append(details, "Every missing rectangle candidate is allowed by the original clues; prior logical eliminations do not block the uniqueness argument.")
					}
					return commitEliminations(s, name, "Avoid a second solution obtained by swapping the two rectangle digits.", targets, details, v)
				}
			}
		}
	}
	return Step{}, false
}

func ApplyAvoidableRectangle(s *Sudoku) (Step, bool) {
	if !s.initialKnown {
		return Step{}, false
	}
	for _, rectangle := range rectangles {
		cells := rectangle[:]
		placed := 0
		var digits uint16
		given := false
		for _, c := range cells {
			given = given || s.initialBoard[c.Row][c.Col] != 0
			if d := s.Board[c.Row][c.Col]; d != 0 {
				placed++
				digits |= 1 << d
			}
		}
		if given || placed < 2 || placed > 3 || candidateCount(digits) != 2 {
			continue
		}
		pair := candidateList(digits)
		for _, first := range pair {
			second := pair[0]
			if first == second {
				second = pair[1]
			}
			expected := [4]int{first, second, second, first}
			valid := true
			var targets []CandidateElimination
			var roofs []Cell
			var extras uint16
			for i, c := range cells {
				if d := int(s.Board[c.Row][c.Col]); d != 0 {
					valid = valid && d == expected[i]
				} else {
					valid = valid && cellHasCandidate(s, c, expected[i])
					roofs = append(roofs, c)
					extra := s.Candidates[c.Row][c.Col] &^ digits
					if placed == 2 && candidateCount(extra) != 1 {
						valid = false
					}
					extras |= extra
					if placed == 3 {
						targets = append(targets, CandidateElimination{c.Row, c.Col, expected[i]})
					}
				}
			}
			if !valid {
				continue
			}
			kind := 1
			if placed == 2 {
				if candidateCount(extras) != 1 {
					continue
				}
				kind = 2
				for _, c := range commonCandidatePeers(s, roofs, singleCandidate(extras)) {
					targets = append(targets, CandidateElimination{c.Row, c.Col, singleCandidate(extras)})
				}
			}
			if len(targets) == 0 {
				continue
			}
			if !s.HasUniqueSolution() {
				return Step{}, false
			}
			v := &StepVisual{Sources: append([]Cell(nil), cells...), Digits: pair, Exact: true}
			return commitEliminations(s, fmt.Sprintf("Avoidable Rectangle Type %d", kind), "The previously placed numbers must not complete a swappable rectangle.", targets, []string{"None of these four cells is an original given.", "The original clues have exactly one solution. Completing this two-digit rectangle would allow all four digits to be swapped.", "At least one unfilled corner must use an extra digit."}, v)
		}
	}
	return Step{}, false
}
