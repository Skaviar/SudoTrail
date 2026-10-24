package solver

import "fmt"

// Cell identifies a position. Row and Col are zero-based.
type Cell struct{ Row, Col int }

// A unit is a row, column, or box: nine cells containing digits 1–9.
type unit struct {
	name  string
	cells []Cell
}

// allUnits returns all 27 Sudoku units:
// 9 rows, 9 columns, and 9 boxes.
func allUnits() []unit {
	units := make([]unit, 0, 27)

	for row := 0; row < 9; row++ {
		cells := make([]Cell, 0, 9)
		for col := 0; col < 9; col++ {
			cells = append(cells, Cell{row, col})
		}
		units = append(units, unit{
			name:  fmt.Sprintf("Row %d", row+1),
			cells: cells,
		})
	}

	for col := 0; col < 9; col++ {
		cells := make([]Cell, 0, 9)
		for row := 0; row < 9; row++ {
			cells = append(cells, Cell{row, col})
		}
		units = append(units, unit{
			name:  fmt.Sprintf("Column %d", col+1),
			cells: cells,
		})
	}

	for boxRow := 0; boxRow < 9; boxRow += 3 {
		for boxCol := 0; boxCol < 9; boxCol += 3 {
			cells := make([]Cell, 0, 9)

			for row := boxRow; row < boxRow+3; row++ {
				for col := boxCol; col < boxCol+3; col++ {
					cells = append(cells, Cell{row, col})
				}
			}

			units = append(units, unit{
				name:  fmt.Sprintf("Box %d", boxNumber(boxRow, boxCol)),
				cells: cells,
			})
		}
	}

	return units
}
