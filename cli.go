package main

import (
	"bufio"
	"fmt"
	"github.com/s-threepio/godoku/solver"
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

func PrintStep(step solver.Step, stepNumber int, s *solver.Sudoku) {
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
		printBoard(s, -1, -1)
	} else {
		printBoard(s, step.Row, step.Col)
	}
}

func printBoard(s *solver.Sudoku, highlightRow, highlightCol int) {
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

// Use the menu's reader for step input as well, so buffered input is preserved.
func solveInteractively(s *solver.Sudoku, reader *bufio.Reader) {
	printBoard(s, -1, -1)
	for stepNumber := 1; ; stepNumber++ {
		step, ok := s.SolveStep()
		if !ok {
			break
		}
		PrintStep(step, stepNumber, s)
		fmt.Print("\nPress Enter to continue...")
		if _, err := reader.ReadString('\n'); err != nil {
			return
		}
	}
	if s.IsSolved() {
		fmt.Println("\nPuzzle solved.")
	} else {
		fmt.Println("\nNo supported technique can make further progress.")
	}
}
