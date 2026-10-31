package solver

import (
	"math/rand"
	"strings"
	"testing"
)

func complexPositions(s *Sudoku) (out [27]cellSet) {
	for u, house := range allUnits() {
		for _, c := range candidateCells(s, house, 7) {
			out[u].add(indexCell(c))
		}
	}
	return
}
func houseMask(houses ...int) (out uint32) {
	for _, u := range houses {
		out |= 1 << u
	}
	return
}
func restrictHouseDigit(s *Sudoku, u int, allowed []Cell) {
	for _, c := range allUnits()[u].cells {
		if !containsCell(allowed, c) {
			s.Candidates[c.Row][c.Col] = removeCandidate(s.Candidates[c.Row][c.Col], 7)
		}
	}
}

func TestComplexFishEndoFinsAndCannibalism(t *testing.T) {
	// r15b3/c68b2. R1C8/R1C9 are in both R1 and B3, so
	// they are endo fins. R1C6 is a base cell covered by C6 AND B2.
	s := patternSudoku()
	restrictHouseDigit(s, 0, []Cell{{0, 5}, {0, 7}, {0, 8}})
	restrictHouseDigit(s, 4, []Cell{{4, 5}, {4, 7}})
	restrictHouseDigit(s, 20, []Cell{{0, 7}, {0, 8}, {1, 7}, {2, 7}})
	f := evaluateComplexFish(complexPositions(s), houseMask(0, 4, 20), houseMask(14, 16, 19), 7)
	if f.name != "Finned Franken Swordfish" || f.endo.count() != 2 || !f.targets.has(5) || !f.sources.has(5) {
		t.Fatalf("missing cannibalistic endo-fin proof: %+v", f)
	}
	before := CaptureBoardState(s)
	v, details := complexFishVisual(f)
	step, ok := applyPatternEliminations(s, f.name, "test", 7, f.targets.cells(), details, v)
	if !ok || !strings.Contains(strings.Join(step.Details, " "), "Cannibalistic") {
		t.Fatal("missing explanation")
	}
	assertPatternStep(t, before, s, step, f.name)
	assertDigitEliminationsSound(t, before, step, 7)
	// Pretending overlapping base candidates are ordinary sources would
	// falsely eliminate candidates elsewhere. Targets must see BOTH fins.
	for _, target := range f.targets.cells() {
		for _, fin := range f.fins.cells() {
			if !cellsSeeEachOther(target, fin) {
				t.Fatal("target does not see every fin")
			}
		}
	}
}

func TestComplexFishSearchFindsBoxAndMixedHousePatterns(t *testing.T) {
	for _, transposed := range []bool{false, true} {
		s := patternSudoku()
		// A finned Franken X-Wing, better known as an Empty Rectangle.
		restrictHouseDigit(s, 5, []Cell{{5, 2}, {5, 4}})
		restrictHouseDigit(s, 18, []Cell{{0, 2}, {1, 0}, {2, 2}})
		if transposed {
			transposePattern(s)
		}
		before := CaptureBoardState(s)
		step, ok := ApplyFrankenFish(s, 2)
		if !ok || !strings.Contains(step.Technique, "Franken") {
			t.Fatalf("Franken fish not found: %+v", step)
		}
		assertPatternStep(t, before, s, step, step.Technique)
		assertDigitEliminationsSound(t, before, step, 7)
	}
	// A Mutant X-Wing has a row and column as bases. The evaluator uses
	// exactly the same proof as for a Franken fish, including its fins.
	s := patternSudoku()
	restrictHouseDigit(s, 5, []Cell{{5, 0}, {5, 5}})
	restrictHouseDigit(s, 10, []Cell{{3, 1}, {6, 1}})
	f := evaluateComplexFish(complexPositions(s), houseMask(5, 10), houseMask(14, 21), 7)
	if !strings.Contains(f.name, "Mutant") || f.targets.empty() {
		t.Fatal("Mutant proof missing")
	}
	before := CaptureBoardState(s)
	v, details := complexFishVisual(f)
	step, _ := applyPatternEliminations(s, f.name, "test", 7, f.targets.cells(), details, v)
	assertDigitEliminationsSound(t, before, step, 7)
}

func TestSiameseFishKeepsSeparateFinProofs(t *testing.T) {
	s := patternSudoku()
	restrictHouseDigit(s, 2, []Cell{{2, 1}, {2, 8}})
	restrictHouseDigit(s, 4, []Cell{{4, 1}, {4, 4}, {4, 8}})
	restrictHouseDigit(s, 7, []Cell{{7, 1}, {7, 5}, {7, 8}})
	before := CaptureBoardState(s)
	step, ok := ApplySiameseFish(s, 3)
	if !ok || step.Technique != "Siamese Swordfish" {
		t.Fatalf("missing Siamese fish: %+v", step)
	}
	assertPatternStep(t, before, s, step, step.Technique)
	assertDigitEliminationsSound(t, before, step, 7)
	want := map[CandidateElimination]bool{{6, 4, 7}: false, {3, 5, 7}: false}
	for _, e := range step.Eliminations {
		if _, exists := want[e]; exists {
			want[e] = true
		}
	}
	for e, found := range want {
		if !found {
			t.Fatalf("missing half's elimination: %+v", e)
		}
	}
}

func TestGeneralFishCountingAgainstIndependentPlacements(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	checked := 0
	for trial := 0; trial < 240; trial++ {
		s := patternSudoku()
		// Preserve a known legal placement, while removing arbitrary other
		// candidates. The independent oracle enumerates ALL legal placements.
		for r := range 9 {
			for c := range 9 {
				if c != (r*3+r/3)%9 && rng.Intn(3) == 0 {
					s.Candidates[r][c] = removeCandidate(s.Candidates[r][c], 7)
				}
			}
		}
		for size := 2; size <= 7; size++ {
			permutation := rng.Perm(27)
			var bases, covers uint32
			for i := 0; i < size; i++ {
				bases |= 1 << permutation[i]
				covers |= 1 << permutation[size+i]
			}
			beforePattern := CaptureBoardState(s)
			// Construct arbitrary generalized fish by leaving uncovered and
			// overlapping base candidates only where a target can see them.
			target := rng.Intn(81)
			valid := true
			for i := range 81 {
				baseCount, coverCount := 0, 0
				for _, u := range topology.houses[i] {
					if bases&(1<<u) != 0 {
						baseCount++
					}
					if covers&(1<<u) != 0 {
						coverCount++
					}
				}
				if baseCount == 0 || baseCount == 1 && coverCount > 0 || topology.peers[target].has(i) {
					continue
				}
				c := cellIndex(i)
				if c.Col == (c.Row*3+c.Row/3)%9 {
					valid = false
					break
				}
				s.Candidates[c.Row][c.Col] = removeCandidate(s.Candidates[c.Row][c.Col], 7)
			}
			if !valid {
				beforePattern.Restore(s)
				continue
			}
			positions := complexPositions(s)
			f := evaluateComplexFish(positions, bases, covers, 7)
			if f.targets.empty() {
				beforePattern.Restore(s)
				continue
			}
			before := CaptureBoardState(s)
			v, details := complexFishVisual(f)
			step, _ := applyPatternEliminations(s, f.name, "test", 7, f.targets.cells(), details, v)
			assertDigitEliminationsSound(t, before, step, 7)
			checked++
			beforePattern.Restore(s)
		}
	}
	if checked == 0 {
		t.Fatal("randomized checks did not exercise an elimination")
	}
}
