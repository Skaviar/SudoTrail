package solver

import (
	"fmt"
	"strings"
	"testing"
)

func transposePattern(s *Sudoku) {
	for row := range 9 {
		for col := row + 1; col < 9; col++ {
			s.Board[row][col], s.Board[col][row] = s.Board[col][row], s.Board[row][col]
			s.Candidates[row][col], s.Candidates[col][row] = s.Candidates[col][row], s.Candidates[row][col]
		}
	}
}

func finnedFixture(sashimi, twoFins, transposed bool) *Sudoku {
	s := patternSudoku()
	units := allUnits()
	first := []Cell{{1, 0}, {1, 2}, {1, 5}}
	if sashimi {
		first = []Cell{{1, 0}, {1, 2}}
	}
	if twoFins {
		first = append(first, Cell{1, 1})
	}
	restrictDigit(s, units[1], first, 7)
	restrictDigit(s, units[4], []Cell{{4, 2}, {4, 5}}, 7)
	if transposed {
		transposePattern(s)
	}
	return s
}

// Exhaustively check one digit's legal placements, independently of the
// pattern detector. This proves eliminations preserve every such placement.
func assertDigitEliminationsSound(t *testing.T, before BoardState, step Step, number int) {
	t.Helper()
	var chosen [9]int
	count := 0
	var visit func(int, uint16, uint16)
	visit = func(row int, cols, boxes uint16) {
		if row == 9 {
			count++
			for _, e := range step.Eliminations {
				if e.Number == number && chosen[e.Row] == e.Col {
					t.Fatalf("%s eliminated a legal digit placement: %+v", step.Technique, e)
				}
			}
			return
		}
		for col := range 9 {
			box := row/3*3 + col/3
			if cols&(1<<col) != 0 || boxes&(1<<box) != 0 || !hasCandidate(before.Candidates[row][col], number) {
				continue
			}
			chosen[row] = col
			visit(row+1, cols|1<<col, boxes|1<<box)
		}
	}
	visit(0, 0, 0)
	if count == 0 {
		t.Fatal("fixture has no legal digit placements")
	}
}

func TestFinnedAndSashimiXWing(t *testing.T) {
	for _, sashimi := range []bool{false, true} {
		for _, twoFins := range []bool{false, true} {
			for _, transposed := range []bool{false, true} {
				t.Run(fmt.Sprintf("sashimi=%v/twoFins=%v/transposed=%v", sashimi, twoFins, transposed), func(t *testing.T) {
					s := finnedFixture(sashimi, twoFins, transposed)
					before := CaptureBoardState(s)
					step, ok := ApplyFinnedXWing(s)
					if !ok {
						t.Fatal("finned pattern not found")
					}
					name := "Finned X-Wing"
					if sashimi {
						name = "Sashimi X-Wing"
					}
					assertPatternStep(t, before, s, step, name)
					assertDigitEliminationsSound(t, before, step, 7)
					if len(step.Eliminations) != 2 || len(step.Visual.Highlights) != 1+btoi(twoFins) {
						t.Fatal("incorrect fins or eliminations")
					}
					outside := Cell{7, 5}
					if transposed {
						outside = Cell{5, 7}
					}
					if !hasCandidate(s.Candidates[outside.Row][outside.Col], 7) {
						t.Fatal("removed a cover candidate unable to see the fins")
					}
				})
			}
		}
	}
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestFinnedXWingRejectsSeparatedFinsAndNoOps(t *testing.T) {
	for _, broken := range []bool{true, false} {
		s := finnedFixture(false, false, false)
		if broken {
			s.Candidates[1][7] |= 1 << 7
		} else {
			s.Candidates[0][2] &^= 1 << 7
			s.Candidates[2][2] &^= 1 << 7
		}
		before := CaptureBoardState(s)
		if _, ok := ApplyFinnedXWing(s); ok || CaptureBoardState(s) != before {
			t.Fatal("accepted separated fins or a pattern with no eliminations")
		}
	}
}

func coloringFixture(wrap, transposed bool) *Sudoku {
	s := patternSudoku()
	units := allUnits()
	if wrap {
		restrictDigit(s, units[0], []Cell{{0, 0}, {0, 1}}, 7)
		restrictDigit(s, units[9+1], []Cell{{0, 1}, {1, 1}}, 7)
	} else {
		restrictDigit(s, units[1], []Cell{{1, 6}, {1, 1}}, 7)
		restrictDigit(s, units[18], []Cell{{1, 1}, {2, 0}}, 7)
		restrictDigit(s, units[9], []Cell{{2, 0}, {5, 0}}, 7)
	}
	if transposed {
		transposePattern(s)
	}
	return s
}

func TestSimpleColoringWrapAndTrap(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		for _, transposed := range []bool{false, true} {
			t.Run(fmt.Sprintf("wrap=%v/transposed=%v", wrap, transposed), func(t *testing.T) {
				s := coloringFixture(wrap, transposed)
				before := CaptureBoardState(s)
				step, ok := ApplySimpleColoring(s)
				if !ok {
					t.Fatal("coloring deduction missing")
				}
				name := "Simple Coloring — Color Trap"
				if wrap {
					name = "Simple Coloring — Color Wrap"
				}
				assertPatternStep(t, before, s, step, name)
				assertDigitEliminationsSound(t, before, step, 7)
				if len(step.Visual.Highlights) != len(step.Visual.Sources) {
					t.Fatal("each colored candidate needs an explicit color")
				}
				if wrap && len(step.Eliminations) != 2 {
					t.Fatal("wrap must eliminate its entire conflicting color")
				}
			})
		}
	}
}

func remoteFixture(transposed bool) *Sudoku {
	s := patternSudoku()
	for _, cell := range []Cell{{1, 6}, {1, 1}, {2, 0}, {5, 0}} {
		s.Candidates[cell.Row][cell.Col] = maskOf(7, 8)
	}
	if transposed {
		transposePattern(s)
	}
	return s
}

func TestRemotePairs(t *testing.T) {
	for _, transposed := range []bool{false, true} {
		s := remoteFixture(transposed)
		before := CaptureBoardState(s)
		step, ok := ApplyRemotePairs(s)
		if !ok {
			t.Fatal("remote pairs not found")
		}
		assertPatternStep(t, before, s, step, "Remote Pairs")
		target, sameColor := Cell{5, 6}, Cell{2, 6}
		if transposed {
			target = Cell{6, 5}
			sameColor = Cell{6, 2}
		}
		if s.Candidates[target.Row][target.Col]&maskOf(7, 8) != 0 {
			t.Fatal("target seeing opposite assignments kept pair digits")
		}
		if s.Candidates[sameColor.Row][sameColor.Col]&maskOf(7, 8) != maskOf(7, 8) {
			t.Fatal("target seeing only one assignment lost digits")
		}
		if len(step.Visual.Highlights) != 8 {
			t.Fatal("both digits need alternating color assignments")
		}
	}
}

func TestColoringRejectsBrokenAndConflictingComponents(t *testing.T) {
	s := remoteFixture(false)
	s.Candidates[2][0] |= 1 << 9
	before := CaptureBoardState(s)
	if _, ok := ApplyRemotePairs(s); ok || CaptureBoardState(s) != before {
		t.Fatal("a broken chain became remote pairs")
	}
	var graph [81][]int
	graph[0] = []int{1, 2}
	graph[1] = []int{0, 2}
	graph[2] = []int{0, 1}
	components := twoColorComponents(graph)
	if len(components) != 1 || !components[0].conflict {
		t.Fatal("odd cycle was not rejected")
	}
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyFinnedXWing, ApplySimpleColoring, ApplyRemotePairs} {
		blank, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		before := CaptureBoardState(blank)
		if _, ok := apply(blank); ok || CaptureBoardState(blank) != before {
			t.Fatal("unconstrained puzzle changed")
		}
	}
}
