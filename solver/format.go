package solver

import (
	"fmt"
	"strings"
)

func cellName(row, col int) string {
	return fmt.Sprintf(
		"R%dC%d",
		row+1,
		col+1,
	)
}

func formatList(values []string) string {
	switch len(values) {
	case 0:
		return ""

	case 1:
		return values[0]

	case 2:
		return values[0] + " and " + values[1]
	}

	return strings.Join(values[:len(values)-1], ", ") +
		", and " +
		values[len(values)-1]
}

func boxNumber(row, col int) int {
	return (row/3)*3 + (col / 3) + 1
}
