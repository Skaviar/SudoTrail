package solver

// StepVisual describes the evidence behind a deduction for a visual renderer.
// Unit coordinates and cell coordinates are zero-based.
type StepVisual struct {
	Sources []Cell       `json:"sources"`
	Digits  []int        `json:"digits"`
	Units   []VisualUnit `json:"units"`
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
