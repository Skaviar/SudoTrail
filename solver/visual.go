package solver

import "fmt"

// StepVisual describes the evidence behind a deduction for a visual renderer.
// Unit coordinates and cell coordinates are zero-based.
type StepVisual struct {
	Highlights     []CandidateHighlight   `json:"highlights,omitempty"`
	CandidateLinks []CandidateLink        `json:"candidateLinks,omitempty"`
	Links          []VisualLink           `json:"links,omitempty"`
	Restrictions   []CandidateElimination `json:"restrictions,omitempty"`
	Sources        []Cell                 `json:"sources"`
	Digits         []int                  `json:"digits"`
	Units          []VisualUnit           `json:"units"`
}

// Highlights distinguish fins and the two alternate assignments in coloring.
type CandidateHighlight struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Number int    `json:"number"`
	Role   string `json:"role"` // fin, color-a, or color-b
}

// CandidateLink connects possible digits, rather than already placed numbers.
// A strong link means exactly one endpoint contains Number in Unit; a weak
// link means they cannot both contain it. Elimination links point to removals.
type CandidateLink struct {
	From   Cell       `json:"from"`
	To     Cell       `json:"to"`
	Number int        `json:"number"`
	Unit   VisualUnit `json:"unit"`
	Kind   string     `json:"kind"` // strong, weak, or elimination
}

type VisualUnit struct {
	Kind  string `json:"kind"` // row, column, or box
	Index int    `json:"index"`
}

func describeUnit(u unit) VisualUnit {
	first, last := u.cells[0], u.cells[len(u.cells)-1]
	if first.Row == last.Row {
		return VisualUnit{Kind: "row", Index: first.Row}
	}
	if first.Col == last.Col {
		return VisualUnit{Kind: "column", Index: first.Col}
	}
	return VisualUnit{Kind: "box", Index: first.Row/3*3 + first.Col/3}
}

func intersectionVisual(cells []Cell, number int) *StepVisual {
	first := cells[0]
	units := []VisualUnit{{Kind: "box", Index: first.Row/3*3 + first.Col/3}}
	sameRow, sameCol := true, true
	for _, cell := range cells {
		sameRow = sameRow && cell.Row == first.Row
		sameCol = sameCol && cell.Col == first.Col
	}
	if sameRow {
		units = append(units, VisualUnit{Kind: "row", Index: first.Row})
	}
	if sameCol {
		units = append(units, VisualUnit{Kind: "column", Index: first.Col})
	}
	return &StepVisual{Sources: cells, Digits: []int{number}, Units: units}
}

// VisualLink records a real blocking relationship, not a geometric guess.
// Number is the digit already placed at From that cannot appear at To.
type VisualLink struct {
	From   Cell       `json:"from"`
	To     Cell       `json:"to"`
	Number int        `json:"number"`
	Unit   VisualUnit `json:"unit"`
}

func sharedUnit(a, b Cell) VisualUnit {
	if a.Row == b.Row {
		return VisualUnit{Kind: "row", Index: a.Row}
	}
	if a.Col == b.Col {
		return VisualUnit{Kind: "column", Index: a.Col}
	}
	return VisualUnit{Kind: "box", Index: a.Row/3*3 + a.Col/3}
}

func unitLabel(u VisualUnit) string {
	name := map[string]string{"row": "Row", "column": "Column", "box": "Box"}[u.Kind]
	return fmt.Sprintf("%s %d", name, u.Index+1)
}

// blockingNumber locates an actual placed peer, preferring a row or column
// explanation before a box explanation. Earlier technique eliminations have
// no placed blocker and must not be illustrated with an invented connection.
func blockingNumber(s *Sudoku, target Cell, number int) (VisualLink, bool) {
	for col := range 9 {
		if int(s.Board[target.Row][col]) == number {
			return VisualLink{From: Cell{Row: target.Row, Col: col}, To: target, Number: number, Unit: VisualUnit{Kind: "row", Index: target.Row}}, true
		}
	}
	for row := range 9 {
		if int(s.Board[row][target.Col]) == number {
			return VisualLink{From: Cell{Row: row, Col: target.Col}, To: target, Number: number, Unit: VisualUnit{Kind: "column", Index: target.Col}}, true
		}
	}
	for row := target.Row / 3 * 3; row < target.Row/3*3+3; row++ {
		for col := target.Col / 3 * 3; col < target.Col/3*3+3; col++ {
			if int(s.Board[row][col]) == number {
				return VisualLink{From: Cell{Row: row, Col: col}, To: target, Number: number, Unit: VisualUnit{Kind: "box", Index: target.Row/3*3 + target.Col/3}}, true
			}
		}
	}
	return VisualLink{}, false
}

func addRestriction(s *Sudoku, v *StepVisual, cell Cell, number int) {
	v.Restrictions = append(v.Restrictions, CandidateElimination{Row: cell.Row, Col: cell.Col, Number: number})
	if link, ok := blockingNumber(s, cell, number); ok {
		v.Links = append(v.Links, link)
		for _, source := range v.Sources {
			if source == link.From {
				return
			}
		}
		v.Sources = append(v.Sources, link.From)
	}
}

func hiddenSingleVisual(s *Sudoku, u unit, target Cell, number int) *StepVisual {
	v := &StepVisual{Digits: []int{number}, Units: []VisualUnit{describeUnit(u)}}
	for _, cell := range u.cells {
		if cell != target && s.Board[cell.Row][cell.Col] == 0 && !hasCandidate(s.Candidates[cell.Row][cell.Col], number) {
			addRestriction(s, v, cell, number)
		}
	}
	return v
}

func singleCellVisual(s *Sudoku, target Cell, number int) *StepVisual {
	v := &StepVisual{}
	for digit := 1; digit <= 9; digit++ {
		if digit != number && !hasCandidate(s.Candidates[target.Row][target.Col], digit) {
			addRestriction(s, v, target, digit)
		}
	}
	return v
}
