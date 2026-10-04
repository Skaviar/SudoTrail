package solver

import (
	"bufio"
	"fmt"
	"os"
)

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	Red    = "\033[31m"
	Blue   = "\033[34m"
)

func PrintStep(step Step, stepNumber int, s *Sudoku) {
	fmt.Println()

	fmt.Printf(
		"%s================ STEP %d ================%s\n",
		Bold,
		stepNumber,
		Reset,
	)

	fmt.Printf(
		"%sTechnique:%s %s%s%s\n",
		Bold,
		Reset,
		Cyan,
		step.Technique,
		Reset,
	)

	fmt.Printf(
		"%sExplanation:%s %s%s%s\n",
		Bold,
		Reset,
		Yellow,
		step.Explanation,
		Reset,
	)

	if step.BeforeCandidates != 0 {
		fmt.Printf(
			"Candidates: %v\n",
			CandidateList(step.BeforeCandidates),
		)
	}

	fmt.Println()

	s.PrintHighlighted(step.Row, step.Col)
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
		fmt.Print("| ")

		for col := range 9 {
			value := s.Board[row][col]

			if value == 0 {
				fmt.Print(". ")
			} else if row == highlightRow && col == highlightCol {
				fmt.Printf(
					"%s%d%s ",
					Green,
					value,
					Reset,
				)
			} else {
				fmt.Printf("%d ", value)
			}

			if col == 2 || col == 5 {
				fmt.Print("| ")
			}
		}

		fmt.Println("|")

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
