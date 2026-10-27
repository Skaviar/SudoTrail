package solver

import (
	"fmt"
	"testing"
)

func fishFixture(size int, transposed bool) *Sudoku {
	s := patternSudoku()
	bases := []int{0, 4, 8, 2}[:size]
	if size == 4 {
		// The two rows in the first band must together touch all three boxes,
		// otherwise a simpler pointing deduction is available first.
		bases = []int{0, 2, 4, 8}
	}
	// A ring gives sparse Swordfish/Jellyfish patterns with no smaller fish.
	covers := []int{1, 7, 4, 6}[:size]
	for i, base := range bases {
		for position := range 9 {
			if position == covers[i] || position == covers[(i+1)%size] {
				continue
			}
			row, col := base, position
			if transposed {
				row, col = col, row
			}
			s.Candidates[row][col] = removeCandidate(s.Candidates[row][col], 7)
		}
	}
	return s
}

func TestXWing(t *testing.T) {
	for _, transposed := range []bool{false, true} {
		t.Run(fmt.Sprint(transposed), func(t *testing.T) {
			s := fishFixture(2, transposed)
			before := CaptureBoardState(s)
			step, ok := ApplyBasicFish(s, 2)
			if !ok {
				t.Fatal("X-Wing not found")
			}
			assertPatternStep(t, before, s, step, "X-Wing")
			if len(step.Eliminations) != 14 || len(step.Visual.Sources) != 4 || len(step.Visual.Units) != 4 {
				t.Fatal("incorrect X-Wing sources, units, or eliminations")
			}
			for _, e := range step.Eliminations {
				base, cover := e.Row, e.Col
				if transposed {
					base, cover = cover, base
				}
				if base == 0 || base == 4 || cover != 1 && cover != 7 || e.Number != 7 {
					t.Fatalf("elimination outside fish cover units: %+v", e)
				}
			}
			after := CaptureBoardState(s)
			if _, ok := ApplyBasicFish(s, 2); ok || CaptureBoardState(s) != after {
				t.Fatal("reported a no-op fish")
			}
		})
	}
}

func TestXWingRejectsFinAndFilledSource(t *testing.T) {
	for _, filled := range []bool{false, true} {
		s := fishFixture(2, false)
		if filled {
			s.Board[0][1] = 7
		} else {
			s.Candidates[0][5] |= 1 << 7
		}
		before := CaptureBoardState(s)
		if _, ok := ApplyBasicFish(s, 2); ok || CaptureBoardState(s) != before {
			t.Fatal("accepted an incomplete basic fish")
		}
	}
}

func TestSolveStepFindsXWing(t *testing.T) {
	s := fishFixture(2, false)
	before := CaptureBoardState(s)
	step, ok := s.SolveStep()
	if !ok {
		t.Fatal("pipeline did not find X-Wing")
	}
	assertPatternStep(t, before, s, step, "X-Wing")
}

func TestSwordfishAndJellyfish(t *testing.T) {
	for size := 3; size <= 4; size++ {
		name := map[int]string{3: "Swordfish", 4: "Jellyfish"}[size]
		for _, transposed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/transposed=%v", name, transposed), func(t *testing.T) {
				s := fishFixture(size, transposed)
				before := CaptureBoardState(s)
				step, ok := ApplyBasicFish(s, size)
				if !ok {
					t.Fatal("fish not found")
				}
				assertPatternStep(t, before, s, step, name)
				if len(step.Eliminations) != (9-size)*size || len(step.Visual.Sources) != 2*size || len(step.Visual.Units) != 2*size {
					t.Fatal("incorrect sparse fish")
				}
				// Verify the real ordering also reaches this technique.
				before.Restore(s)
				if size == 4 {
					// Dense intersections avoid a simpler Skyscraper in the sparse
					// ring. The solver should always prefer that simpler move.
					for _, base := range []int{0, 2, 4, 8} {
						for _, cover := range []int{1, 7, 4, 6} {
							row, col := base, cover
							if transposed {
								row, col = col, row
							}
							s.Candidates[row][col] |= 1 << 7
						}
					}
					before = CaptureBoardState(s)
				}
				step, ok = s.SolveStep()
				if !ok {
					t.Fatal("pipeline missed fish")
				}
				assertPatternStep(t, before, s, step, name)
			})
		}
	}
}

func TestFishRejectsTooManyCoversAndInvalidSizes(t *testing.T) {
	for size := 2; size <= 4; size++ {
		s := fishFixture(size, false)
		s.Candidates[0][0] |= 1 << 7 // A fin outside the selected cover columns.
		before := CaptureBoardState(s)
		if _, ok := ApplyBasicFish(s, size); ok || CaptureBoardState(s) != before {
			t.Fatalf("size %d accepted too many cover columns", size)
		}
	}
	for _, size := range []int{-1, 0, 1, 5, 9} {
		s := fishFixture(2, false)
		before := CaptureBoardState(s)
		if _, ok := ApplyBasicFish(s, size); ok || CaptureBoardState(s) != before {
			t.Fatal("accepted unsupported fish size")
		}
	}
}
