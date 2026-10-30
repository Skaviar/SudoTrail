package solver

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// Both the fin-free pattern and a pattern with a true fin have legal digit
// placements. They differ by swapping r2c3/r7c1 with r2c1/r7c3.
func largerFinnedFixture(size int, sparse, sashimi, multipleFins, transposed bool) *Sudoku {
	s := patternSudoku()
	bases := []int{1, 0, 4, 7, 3, 5, 8}[:size]
	placement := []int{3, 2, 6, 1, 4, 7, 0, 5, 8}
	var covers []int
	for _, base := range bases {
		covers = append(covers, placement[base])
	}
	for i, base := range bases {
		var cells []Cell
		for _, cover := range covers {
			if sparse && cover != placement[base] && cover != covers[(i+1)%size] {
				continue
			}
			if sashimi && base == 1 && cover != placement[base] {
				continue
			}
			cells = append(cells, Cell{base, cover})
		}
		if base == 1 || multipleFins && base == 0 {
			cells = append(cells, Cell{base, 0})
		}
		restrictDigit(s, allUnits()[base], cells, 7)
	}
	if transposed {
		transposePattern(s)
	}
	return s
}

func TestLargerFinnedAndSashimiFish(t *testing.T) {
	names := map[int]string{3: "Swordfish", 4: "Jellyfish", 5: "Squirmbag", 6: "Whale", 7: "Leviathan"}
	for size := 3; size <= 7; size++ {
		for _, sparse := range []bool{false, true} {
			for _, sashimi := range []bool{false, true} {
				for _, multiple := range []bool{false, true} {
					for _, transposed := range []bool{false, true} {
						name := fmt.Sprintf("size=%d/sparse=%v/sashimi=%v/multiple=%v/transposed=%v", size, sparse, sashimi, multiple, transposed)
						t.Run(name, func(t *testing.T) {
							s := largerFinnedFixture(size, sparse, sashimi, multiple, transposed)
							before := CaptureBoardState(s)
							step, ok := ApplyFinnedFish(s, size)
							if !ok {
								t.Fatal("finned fish not found")
							}
							technique := "Finned " + names[size]
							if sashimi {
								technique = "Sashimi " + names[size]
							}
							// A sparse large grid can have another valid fish in
							// the other orientation, found before the planted one.
							if sparse && transposed && size >= 5 {
								if step.Technique != "Finned "+names[size] && step.Technique != "Sashimi "+names[size] {
									t.Fatal("incorrect fish size or family")
								}
								technique = step.Technique
							}
							assertPatternStep(t, before, s, step, technique)
							assertDigitEliminationsSound(t, before, step, 7)
							assertFinnedEvidence(t, before, step, size)
							box := step.Visual.Units[len(step.Visual.Units)-1].Index
							for row := range 9 {
								for col := range 9 {
									if row/3*3+col/3 != box && s.Candidates[row][col] != before.Candidates[row][col] {
										t.Fatal("changed candidates outside the fin box")
									}
								}
							}
						})
					}
				}
			}
		}
	}
}

func assertFinnedEvidence(t *testing.T, before BoardState, step Step, size int) {
	t.Helper()
	v := step.Visual
	if len(v.Units) != 2*size+1 || len(v.Highlights) == 0 || len(step.Details) < 5 {
		t.Fatal("missing base units, cover units, fin box, or explanation")
	}
	units := allUnits()
	baseKind, coverKind := "row", "column"
	baseOffset, coverOffset := 0, 9
	if v.Units[0].Kind == "column" {
		baseKind, coverKind = coverKind, baseKind
		baseOffset, coverOffset = coverOffset, baseOffset
	}
	for _, fin := range v.Highlights {
		cell := Cell{fin.Row, fin.Col}
		if fin.Role != "fin" || fin.Number != 7 || !hasCandidate(before.Candidates[fin.Row][fin.Col], 7) || !containsCell(v.Sources, cell) {
			t.Fatal("fin is not an actual source candidate")
		}
		for _, u := range v.Units {
			if u.Kind == coverKind && containsCell(units[coverOffset+u.Index].cells, cell) {
				t.Fatal("highlighted fin was covered")
			}
		}
		for _, e := range step.Eliminations {
			if !cellsSeeEachOther(cell, Cell{e.Row, e.Col}) {
				t.Fatal("target cannot see every fin")
			}
		}
	}
	for _, e := range step.Eliminations {
		target := Cell{e.Row, e.Col}
		covered := false
		for _, u := range v.Units {
			if u.Kind == baseKind && containsCell(units[baseOffset+u.Index].cells, target) {
				t.Fatal("eliminated a base candidate")
			}
			if u.Kind == coverKind && containsCell(units[coverOffset+u.Index].cells, target) {
				covered = true
			}
		}
		if !covered || e.Number != 7 {
			t.Fatal("elimination is not a fish-digit cover candidate")
		}
	}
}

func TestFinnedFishRejectsBrokenPatternsAndNoOps(t *testing.T) {
	for size := 3; size <= 7; size++ {
		for _, kind := range []string{"separated-fins", "no-targets", "no-fins", "single-base"} {
			// Removing targets at sizes 6–7 exposes a different, productive
			// complementary fish; use smaller fixtures for the no-op check.
			if kind == "no-targets" && size >= 6 {
				continue
			}
			t.Run(fmt.Sprintf("size=%d/%s", size, kind), func(t *testing.T) {
				s := largerFinnedFixture(size, false, false, false, false)
				switch kind {
				case "separated-fins":
					s.Candidates[4][0] |= 1 << 7
				case "no-targets":
					for col := range 3 {
						s.Candidates[2][col] = removeCandidate(s.Candidates[2][col], 7)
					}
				case "no-fins":
					s.Candidates[1][0] = removeCandidate(s.Candidates[1][0], 7)
				case "single-base":
					s.Candidates[1][0] = maskOf(7)
					for col := 1; col < 9; col++ {
						s.Candidates[1][col] = removeCandidate(s.Candidates[1][col], 7)
					}
				}
				before := CaptureBoardState(s)
				if step, ok := ApplyFinnedFish(s, size); ok || CaptureBoardState(s) != before {
					t.Fatalf("accepted broken pattern or no-op: %+v", step)
				}
			})
		}
	}
	for _, size := range []int{-1, 0, 1, 8, 9} {
		s := patternSudoku()
		before := CaptureBoardState(s)
		if _, ok := ApplyFinnedFish(s, size); ok || CaptureBoardState(s) != before {
			t.Fatal("invalid fish size changed the puzzle")
		}
	}
	blank, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
	for size := 2; size <= 7; size++ {
		before := CaptureBoardState(blank)
		if _, ok := ApplyFinnedFish(blank, size); ok || CaptureBoardState(blank) != before {
			t.Fatal("unconstrained puzzle changed")
		}
	}
}

func TestSashimiFishWithCoverSingleOrSmallerCore(t *testing.T) {
	for _, size := range []int{3, 4} {
		for _, transposed := range []bool{false, true} {
			s := largerFinnedFixture(size, false, false, false, false)
			// Every base keeps at least two core candidates. Size 3 has a
			// cover with one location; size 4 has a proper X-Wing core.
			for _, row := range []int{0, 1} {
				for _, col := range []int{4, 5} {
					s.Candidates[row][col] = removeCandidate(s.Candidates[row][col], 7)
				}
			}
			if transposed {
				transposePattern(s)
			}
			before := CaptureBoardState(s)
			step, ok := ApplyFinnedFish(s, size)
			if !ok {
				t.Fatal("reducible fin-free core was not found")
			}
			name := map[int]string{3: "Sashimi Swordfish", 4: "Sashimi Jellyfish"}[size]
			assertPatternStep(t, before, s, step, name)
			assertDigitEliminationsSound(t, before, step, 7)
		}
	}
}

// Candidate removals are checked against all legal placements, not against a
// second implementation of the fish detector. Every trial keeps a known legal
// placement, so a successful assertion cannot rely on an impossible fixture.
func TestFinnedFishRandomCandidateEliminations(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	placement := []int{3, 2, 6, 1, 4, 7, 0, 5, 8}
	found := 0
	for range 120 {
		s := patternSudoku()
		for row := range 9 {
			for col := range 9 {
				if col != placement[row] && rng.Intn(100) < 60 {
					s.Candidates[row][col] = removeCandidate(s.Candidates[row][col], 7)
				}
			}
		}
		for size := 2; size <= 7; size++ {
			copy := *s
			before := CaptureBoardState(&copy)
			if step, ok := ApplyFinnedFish(&copy, size); ok {
				found++
				assertPatternStep(t, before, &copy, step, step.Technique)
				assertDigitEliminationsSound(t, before, step, 7)
			}
		}
	}
	if found == 0 {
		t.Fatal("randomized check exercised no deductions")
	}
}
