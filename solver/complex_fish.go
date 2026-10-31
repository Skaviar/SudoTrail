package solver

import (
	"fmt"
	"math/bits"
	"sort"
)

type complexFish struct {
	digit                        int
	bases, covers                uint32
	sources, fins, endo, targets cellSet
	name                         string
}

func fishFamily(bases, covers uint32) string {
	rowMask, colMask, boxMask := uint32(511), uint32(511<<9), uint32(511<<18)
	if bases&boxMask == 0 && covers&boxMask == 0 && (bases&rowMask == 0 || bases&colMask == 0) && (covers&rowMask == 0 || covers&colMask == 0) {
		return ""
	}
	if bases&rowMask != 0 && bases&colMask != 0 || covers&rowMask != 0 && covers&colMask != 0 {
		return "Mutant"
	}
	return "Franken"
}

// evaluateComplexFish is the counting proof, shared by every fish shape.
// With all fins false, N distinct base placements must fill N cover houses.
// An outside candidate using a cover, or a base candidate using TWO covers,
// leaves too little room. Overlapping base candidates are endo fins: treating
// them as ordinary base placements would invalidate this counting argument.
func evaluateComplexFish(positions [27]cellSet, bases, covers uint32, digit int) complexFish {
	f := complexFish{digit: digit, bases: bases, covers: covers}
	var covered, twiceCovered cellSet
	for u := range 27 {
		if bases&(1<<u) != 0 {
			f.endo = f.endo.or(f.sources.and(positions[u]))
			f.sources = f.sources.or(positions[u])
		}
		if covers&(1<<u) != 0 {
			twiceCovered = twiceCovered.or(covered.and(positions[u]))
			covered = covered.or(positions[u])
		}
	}
	f.fins = f.sources.minus(covered).or(f.endo)
	f.targets = covered.minus(f.sources).or(twiceCovered.and(f.sources))
	if !f.fins.empty() {
		f.targets = f.targets.and(commonPeers(f.fins))
	}
	f.targets = f.targets.minus(f.fins)
	family := fishFamily(bases, covers)
	f.name = family + " " + fishNames[bits.OnesCount32(bases)]
	if !f.fins.empty() {
		f.name = "Finned " + f.name
	}
	return f
}

func complexFishVisual(f complexFish) (*StepVisual, []string) {
	v := &StepVisual{Sources: f.sources.cells(), Digits: []int{f.digit}, Exact: true}
	var bases, covers []string
	for u := range 27 {
		if f.bases&(1<<u) != 0 {
			v.Units = append(v.Units, visualHouse(u))
			bases = append(bases, unitLabel(visualHouse(u)))
		}
		if f.covers&(1<<u) != 0 {
			v.Units = append(v.Units, visualHouse(u))
			covers = append(covers, unitLabel(visualHouse(u)))
		}
	}
	for _, c := range f.fins.cells() {
		v.Highlights = append(v.Highlights, CandidateHighlight{c.Row, c.Col, f.digit, "fin"})
	}
	details := []string{fmt.Sprintf("Base houses: %s. Cover houses: %s.", formatList(bases), formatList(covers)), "With every fin false, each base house contributes a distinct placement and all cover houses are occupied.", "An eliminated candidate sees every fin; if it were true, all fins would be false and the cover houses could not hold all required placements."}
	if !f.endo.empty() {
		details = append(details, fmt.Sprintf("Endo fins at %s belong to multiple base houses and must be treated as fins.", formatList(cellNames(f.endo.cells()))))
	}
	if !f.targets.and(f.sources).empty() {
		details = append(details, "Cannibalistic elimination: a target inside the bases occupies two cover houses at once.")
	}
	linkEliminations(v, f.targets.cells(), f.fins.cells(), f.digit)
	return v, details
}

// ApplyComplexFish supports sizes 2..7, boxes on either side, endo fins and
// cannibalism. Finding a proof is bounded (not exhaustive) to keep browser and
// multi-million-puzzle runs responsive. The bound limits discovery, never the
// validity of a returned deduction. No puzzle completions are enumerated.
func ApplyComplexFish(s *Sudoku, size int) (Step, bool) {
	return applyComplexFish(s, size, "")
}

func ApplyFrankenFish(s *Sudoku, size int) (Step, bool) {
	return applyComplexFish(s, size, "Franken")
}

func ApplyMutantFish(s *Sudoku, size int) (Step, bool) {
	return applyComplexFish(s, size, "Mutant")
}

func applyComplexFish(s *Sudoku, size int, family string) (Step, bool) {
	if size < 2 || size > 7 {
		return Step{}, false
	}
	for d := 1; d <= 9; d++ {
		all := digitCells(s, d)
		var positions [27]cellSet
		var eligible []int
		for u, house := range allUnits() {
			for _, c := range candidateCells(s, house, d) {
				positions[u].add(indexCell(c))
			}
			if n := positions[u].count(); n >= 2 && n <= size+2 {
				eligible = append(eligible, u)
			}
		}
		if len(eligible) < size {
			continue
		}
		sort.SliceStable(eligible, func(i, j int) bool { return positions[eligible[i]].count() < positions[eligible[j]].count() })
		budget := 3000 // Cover-tree nodes per digit and fish size.
		var found complexFish
		var chooseBases func(int, int, uint32, cellSet, cellSet) bool
		chooseBases = func(start, left int, bases uint32, union, endo cellSet) bool {
			if budget <= 0 {
				return false
			}
			if left > 0 {
				for j := start; j <= len(eligible)-left; j++ {
					u := eligible[j]
					overlaps := endo.or(union.and(positions[u]))
					if !overlaps.empty() && commonPeers(overlaps).and(all).empty() {
						continue
					}
					if chooseBases(j+1, left-1, bases|1<<u, union.or(positions[u]), overlaps) {
						return true
					}
					if budget <= 0 {
						break
					}
				}
				return false
			}
			budget--
			potential := all
			if !endo.empty() {
				potential = potential.and(commonPeers(endo))
			}
			for _, target := range potential.cells() {
				ti := indexCell(target)
				// Assuming the target true makes all its peers false. These
				// are the ONLY base candidates permitted to become fins.
				required := union.minus(topology.peers[ti])
				needTarget := 1
				if union.has(ti) {
					needTarget = 2
				}
				var cover func(uint32, cellSet, int) bool
				cover = func(covers uint32, covered cellSet, targetCount int) bool {
					budget--
					if budget < 0 {
						return false
					}
					count := bits.OnesCount32(covers)
					if count == size {
						if !required.minus(covered).empty() || targetCount < needTarget || fishFamily(bases, covers) == "" || family != "" && fishFamily(bases, covers) != family {
							return false
						}
						f := evaluateComplexFish(positions, bases, covers, d)
						if f.targets.has(ti) {
							found = f
							return true
						}
						return false
					}
					remaining := required.minus(covered)
					var options []int
					if !remaining.empty() {
						options = append(options, topology.houses[remaining.first()][:]...)
					} else {
						// Cover every required candidate, then complete the
						// proof with houses containing the proposed target.
						options = append(options, topology.houses[ti][:]...)
					}
					for _, u := range options {
						if (bases|covers)&(1<<u) != 0 || positions[u].empty() {
							continue
						}
						nextCount := targetCount
						if positions[u].has(ti) {
							nextCount++
						}
						if cover(covers|1<<u, covered.or(positions[u]), nextCount) {
							return true
						}
					}
					return false
				}
				if cover(0, cellSet{}, 0) {
					return true
				}
				if budget <= 0 {
					break
				}
			}
			return false
		}
		if chooseBases(0, size, 0, cellSet{}, cellSet{}) {
			if step, ok := siameseComplexFishStep(s, positions, found); ok {
				return step, true
			}
			v, details := complexFishVisual(found)
			return applyPatternEliminations(s, found.name, "The base and cover houses leave no room for these candidates.", d, found.targets.cells(), details, v)
		}
	}
	return Step{}, false
}

func siameseComplexFishStep(s *Sudoku, positions [27]cellSet, first complexFish) (Step, bool) {
	if first.fins.empty() {
		return Step{}, false
	}
	for remove := range 27 {
		if first.covers&(1<<remove) == 0 {
			continue
		}
		for add := range 27 {
			if (first.bases|first.covers)&(1<<add) != 0 || positions[add].empty() {
				continue
			}
			covers := first.covers&^(1<<remove) | 1<<add
			if fishFamily(first.bases, covers) != fishFamily(first.bases, first.covers) {
				continue
			}
			second := evaluateComplexFish(positions, first.bases, covers, first.digit)
			if second.fins.empty() || second.targets.minus(first.targets).empty() || first.targets.minus(second.targets).empty() {
				continue
			}
			v, details := complexFishVisual(first)
			other, more := complexFishVisual(second)
			v.Highlights = append(v.Highlights, other.Highlights...)
			v.Units = append(v.Units, other.Units...)
			v.CandidateLinks = append(v.CandidateLinks, other.CandidateLinks...)
			details = append(details, "Second fish: a different cover house and its own fin proof.")
			details = append(details, more...)
			return applyPatternEliminations(s, "Siamese "+first.name, "Two complex fish on the same sources give separate, complementary eliminations.", first.digit, first.targets.or(second.targets).cells(), details, v)
		}
	}
	return Step{}, false
}

// Siamese fish share the same bases and source cells, but differ in one cover
// house. Each half has its own fins and proof; their fin sets are never merged
// into a false claim that every target sees every fin of both halves.
func ApplySiameseFish(s *Sudoku, size int) (Step, bool) {
	if size < 2 || size > 7 {
		return Step{}, false
	}
	for d := 1; d <= 9; d++ {
		for direction := range 2 {
			p := fishPositions(s, d, direction)
			var positions [27]cellSet
			for u, house := range allUnits() {
				for _, c := range candidateCells(s, house, d) {
					positions[u].add(indexCell(c))
				}
			}
			for _, selection := range fishSelections[size] {
				var bases uint32
				var union uint16
				valid := true
				for b := range 9 {
					if selection&(1<<b) != 0 {
						valid = valid && candidateCount(p[b]) >= 2
						union |= p[b]
						bases |= 1 << (b + direction*9)
					}
				}
				if !valid || candidateCount(union) <= size || candidateCount(union) > size+3 {
					continue
				}
				var halves []complexFish
				for _, coverSelection := range fishSelections[size] {
					if coverSelection&^union != 0 {
						continue
					}
					var covers uint32
					for c := range 9 {
						if coverSelection&(1<<c) != 0 {
							covers |= 1 << (c + (1-direction)*9)
						}
					}
					f := evaluateComplexFish(positions, bases, covers, d)
					if f.fins.empty() || f.targets.empty() {
						continue
					}
					for _, previous := range halves {
						if bits.OnesCount32(previous.covers^covers) != 2 || f.targets.minus(previous.targets).empty() || previous.targets.minus(f.targets).empty() {
							continue
						}
						v, details := complexFishVisual(previous)
						other, more := complexFishVisual(f)
						v.Highlights = append(v.Highlights, other.Highlights...)
						v.Units = append(v.Units, other.Units...)
						v.CandidateLinks = append(v.CandidateLinks, other.CandidateLinks...)
						details = append(details, "Second fish, with a different cover house:")
						details = append(details, more...)
						return applyPatternEliminations(s, "Siamese "+fishNames[size], "Two finned fish on the same source cells produce complementary eliminations.", d, previous.targets.or(f.targets).cells(), details, v)
					}
					halves = append(halves, f)
				}
			}
		}
	}
	return Step{}, false
}
