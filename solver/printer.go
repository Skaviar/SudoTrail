package solver

import (
	"bufio"
	"fmt"
	"os"
)

const (
	Reset = "\033[0m"
	Bold  = "\033[1m"

	// Explicitly set normal text colors.
	// Do NOT rely on Reset because your terminal itself is green.
	Normal = "\033[37m"

	Cyan   = "\033[36m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Red    = "\033[31m"

	// Newly solved cell.
	LightGreen = "\033[92m"
)

func PrintStep(step Step, stepNumber int, s *Sudoku) {
	fmt.Println()

	fmt.Printf(
		"%s%s================ STEP %d ================%s\n",
		Bold,
		Normal,
		stepNumber,
		Reset,
	)

	fmt.Printf(
		"%sTechnique:%s %s%s%s\n",
		Normal,
		Reset,
		Cyan,
		step.Technique,
		Reset,
	)

	fmt.Printf(
		"\n%s%s%s\n",
		Yellow,
		step.Explanation,
		Reset,
	)

	for _, detail := range step.Details {
		fmt.Printf(
			"%s  • %s%s\n",
			Normal,
			detail,
			Reset,
		)
	}

	fmt.Println()
	if len(step.Eliminations) > 0 {
		s.Print()
	} else {
		s.PrintHighlighted(step.Row, step.Col)
	}
}

func (s *Sudoku) Print() {
	s.printBoard(-1, -1)
}

func (s *Sudoku) PrintHighlighted(row, col int) {
	s.printBoard(row, col)
}

func (s *Sudoku) printBoard(highlightRow, highlightCol int) {
	fmt.Println("+-------+-------+-------+")

	for row := range 9 {
		fmt.Printf("%s| ", Normal)

		for col := range 9 {
			value := s.Board[row][col]

			switch {
			case value == 0:
				fmt.Print(". ")

			case row == highlightRow && col == highlightCol:
				fmt.Printf(
					"%s%d%s ",
					LightGreen,
					value,
					Normal,
				)

			default:
				fmt.Printf(
					"%s%d%s ",
					Normal,
					value,
					Normal,
				)
			}

			if col == 2 || col == 5 {
				fmt.Printf("%s| %s", Normal, Normal)
			}
		}

		fmt.Printf("%s|%s\n", Normal, Reset)

		if row == 2 || row == 5 {
			fmt.Println("+-------+-------+-------+")
		}
	}

	fmt.Println("+-------+-------+-------+")
}

func WaitForEnter() {
	fmt.Printf(
		"\n%sPress Enter to continue...%s",
		Blue,
		Reset,
	)

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}
