
package solver

import (
	"fmt"
	"math/bits"
	"strings"
)

type subsetUnit struct {
	name  string
	cells [][2]int
}

// FindHiddenSubset finds one Hidden Pair (size 2),
// Hidden Triple (size 3), or Hidden Quadruple (size 4).
func FindHiddenSubset(s *Sudoku, size int) (Step, bool) {
	if size < 2 || size > 4 {
		return Step{}, false
	}

	technique := map[int]string{
		2: "Hidden Pair",
		3: "Hidden Triple",
		4: "Hidden Quadruple",
	}[size]

	for _, unit := range getSubsetUnits() {
		// Try every combination of 'size' digits.
		for digitMask := uint16(1); digitMask <= AllCandidates; digitMask++ {
			// Candidate bit 0 is unused.
			if digitMask&1 != 0 || bits.OnesCount16(digitMask) != size {
				continue
			}

			var subsetCells [][2]int
			digitFound := make(map[int]bool)

			// Find all cells in this unit that can contain
			// at least one of the selected digits.
			for _, cell := range unit.cells {
				row, col := cell[0], cell[1]

				if s.Board[row][col] != 0 {
					continue
				}

				cellMask := s.Candidates[row][col]
				if cellMask&digitMask == 0 {
					continue
				}

				subsetCells = append(subsetCells, cell)

				for digit := 1; digit <= 9; digit++ {
					if digitMask&(1<<digit) != 0 &&
						hasCandidate(cellMask, digit) {
						digitFound[digit] = true
					}
				}
			}

			// Every selected digit must have at least one
			// possible location in this unit.
			allDigitsFound := true
			for digit := 1; digit <= 9; digit++ {
				if digitMask&(1<<digit) != 0 && !digitFound[digit] {
					allDigitsFound = false
					break
				}
			}

			if !allDigitsFound || len(subsetCells) != size {
				continue
			}

			// These digits are confined to exactly 'size'
			// cells. Remove all other candidates from those cells.
			var eliminations []CandidateElimination

			for _, cell := range subsetCells {
				row, col := cell[0], cell[1]

				for digit := 1; digit <= 9; digit++ {
					if digitMask&(1<<digit) != 0 {
						continue
					}

					if hasCandidate(s.Candidates[row][col], digit) {
						s.Candidates[row][col] =
							removeCandidate(s.Candidates[row][col], digit)

						eliminations = append(eliminations, CandidateElimination{
							Row:    row,
							Col:    col,
							Number: digit,
						})
					}
				}
			}

			// Ignore subsets that don't actually eliminate anything.
			if len(eliminations) == 0 {
				continue
			}

			digits := maskDigits(digitMask)
			cellLabels := cellNames(subsetCells)

			details := []string{
				fmt.Sprintf(
					"In %s, digits %s can only appear in %s.",
					unit.name,
					formatList(digits),
					formatList(cellLabels),
				),
				fmt.Sprintf(
					"These %d cells must contain those %d digits, so remove all other candidates from them.",
					size, size,
				),
			}

			for _, elimination := range eliminations {
				details = append(details, fmt.Sprintf(
					"Eliminate %d from %s.",
					elimination.Number,
					cellName(elimination.Row, elimination.Col),
				))
			}

			return Step{
				Technique:    technique,
				Explanation:  fmt.Sprintf("%s found in %s.", technique, unit.name),
				Details:      details,
				Eliminations: eliminations,
			}, true
		}
	}

	return Step{}, false
}

// getSubsetUnits returns all 27 Sudoku units:
// 9 rows, 9 columns, and 9 boxes.
func getSubsetUnits() []subsetUnit {
	units := make([]subsetUnit, 0, 27)

	for row := 0; row < 9; row++ {
		cells := make([][2]int, 0, 9)
		for col := 0; col < 9; col++ {
			cells = append(cells, [2]int{row, col})
		}
		units = append(units, subsetUnit{
			name:  fmt.Sprintf("Row %d", row+1),
			cells: cells,
		})
	}

	for col := 0; col < 9; col++ {
		cells := make([][2]int, 0, 9)
		for row := 0; row < 9; row++ {
			cells = append(cells, [2]int{row, col})
		}
		units = append(units, subsetUnit{
			name:  fmt.Sprintf("Column %d", col+1),
			cells: cells,
		})
	}

	for boxRow := 0; boxRow < 9; boxRow += 3 {
		for boxCol := 0; boxCol < 9; boxCol += 3 {
			cells := make([][2]int, 0, 9)

			for row := boxRow; row < boxRow+3; row++ {
				for col := boxCol; col < boxCol+3; col++ {
					cells = append(cells, [2]int{row, col})
				}
			}

			units = append(units, subsetUnit{
				name:  fmt.Sprintf("Box %d", boxNumber(boxRow, boxCol)),
				cells: cells,
			})
		}
	}

	return units
}

func maskDigits(mask uint16) []string {
	var digits []string

	for digit := 1; digit <= 9; digit++ {
		if mask&(1<<digit) != 0 {
			digits = append(digits, fmt.Sprint(digit))
		}
	}

	return digits
}

// Keep this helper if you don't already have an equivalent
// in your solver package.
func subsetCellNames(cells [][2]int) string {
	names := make([]string, 0, len(cells))

	for _, cell := range cells {
		names = append(names, cellName(cell[0], cell[1]))
	}

	return strings.Join(names, ", ")
}
