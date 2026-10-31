package solver

import "math/bits"

// cellSet is an 81-bit set. Unlike a candidate mask, its bits name CELLS.
type cellSet struct{ lo, hi uint64 }

func (a cellSet) or(b cellSet) cellSet    { return cellSet{a.lo | b.lo, a.hi | b.hi} }
func (a cellSet) and(b cellSet) cellSet   { return cellSet{a.lo & b.lo, a.hi & b.hi} }
func (a cellSet) minus(b cellSet) cellSet { return cellSet{a.lo &^ b.lo, a.hi &^ b.hi} }
func (a cellSet) empty() bool             { return a.lo|a.hi == 0 }
func (a cellSet) count() int              { return bits.OnesCount64(a.lo) + bits.OnesCount64(a.hi) }
func (a cellSet) has(i int) bool {
	if i < 64 {
		return a.lo&(uint64(1)<<i) != 0
	}
	return a.hi&(uint64(1)<<(i-64)) != 0
}
func (a *cellSet) add(i int) {
	if i < 64 {
		a.lo |= uint64(1) << i
	} else {
		a.hi |= uint64(1) << (i - 64)
	}
}
func (a cellSet) first() int {
	if a.lo != 0 {
		return bits.TrailingZeros64(a.lo)
	}
	return 64 + bits.TrailingZeros64(a.hi)
}
func (a cellSet) cells() []Cell {
	var out []Cell
	for !a.empty() {
		i := a.first()
		out = append(out, cellIndex(i))
		a = a.minus(singleCellSet(i))
	}
	return out
}
func singleCellSet(i int) cellSet { var out cellSet; out.add(i); return out }
func cellIndex(i int) Cell        { return Cell{i / 9, i % 9} }
func indexCell(c Cell) int        { return c.Row*9 + c.Col }

// Shared topology is initialized once and never mutated by puzzle workers.
var topology = func() (t struct {
	units  [27]cellSet
	peers  [81]cellSet
	houses [81][3]int
}) {
	for i := range 81 {
		c := cellIndex(i)
		t.houses[i] = [3]int{c.Row, 9 + c.Col, 18 + c.Row/3*3 + c.Col/3}
		for _, u := range t.houses[i] {
			t.units[u].add(i)
		}
		for j := range 81 {
			if cellsSeeEachOther(c, cellIndex(j)) {
				t.peers[i].add(j)
			}
		}
	}
	return
}()

func digitCells(s *Sudoku, d int) cellSet {
	var out cellSet
	for i := range 81 {
		if cellHasCandidate(s, cellIndex(i), d) {
			out.add(i)
		}
	}
	return out
}
func commonPeers(cells cellSet) cellSet {
	out := cellSet{^uint64(0), (1 << 17) - 1}
	if cells.empty() {
		return cellSet{}
	}
	for !cells.empty() {
		i := cells.first()
		out = out.and(topology.peers[i])
		cells = cells.minus(singleCellSet(i))
	}
	return out
}
func visualHouse(i int) VisualUnit {
	return VisualUnit{Kind: []string{"row", "column", "box"}[i/9], Index: i % 9}
}
