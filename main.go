package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/s-threepio/godoku/solver"
)

func main() {
	file, err := os.Open("sudoku-3m.csv")
	if err != nil {
		slog.Error("readfile err", slog.Any("error", err))
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var puzzles []solver.Puzzle

	for scanner.Scan() {
		appendPuzzle(&puzzles, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error reading puzzles", slog.Any("error", err))
		return
	}

	if len(puzzles) == 0 {
		fmt.Println("No puzzles found.")
		return
	}

	reader := bufio.NewReader(os.Stdin)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		fmt.Println("\n=== Godoku ===")
		fmt.Println("1. Enter a Sudoku")
		fmt.Println("2. Solve a random database puzzle")
		fmt.Println("3. Solve as many database puzzles as possible")
		fmt.Println("0. Exit")
		fmt.Print("\nChoose an option: ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		switch strings.TrimSpace(choice) {
		case "1":
			puzzle, ok := readSudoku(reader)
			if !ok {
				continue
			}

			s := &solver.Sudoku{}
			s.InitializeSudoku(puzzle)
			s.Print()
			s.Solve()
			fmt.Println("\nAfter solving:")
			s.Print()

		case "2":
			puzzle := puzzles[rng.Intn(len(puzzles))]
			fmt.Printf("\nPuzzle ID: %s | Difficulty: %s\n",
				puzzle.Id, puzzle.Difficulty)

			s := &solver.Sudoku{}
			s.InitializeSudoku(puzzle)
			s.Print()
			s.Solve()
			fmt.Println("\nAfter solving:")
			s.Print()

		case "3":
			runBenchmark(puzzles)

		case "0":
			return

		default:
			fmt.Println("Invalid option. Choose 0, 1, 2, or 3.")
		}
	}
}

func readSudoku(reader *bufio.Reader) (solver.Puzzle, bool) {
	fmt.Println("\nEnter your Sudoku row by row.")
	fmt.Println("Use 1-9 for clues and 0 or . for empty cells.")
	fmt.Println("Enter exactly 9 characters per row.")

	var rows []string

	for row := 0; row < 9; row++ {
		for {
			fmt.Printf("Row %d: ", row+1)

			line, err := reader.ReadString('\n')
			if err != nil {
				return solver.Puzzle{}, false
			}

			line = strings.TrimSpace(line)
			line = strings.ReplaceAll(line, " ", "")

			if len(line) != 9 {
				fmt.Println("Please enter exactly 9 characters.")
				continue
			}

			valid := true
			for _, ch := range line {
				if ch != '.' && ch != '0' && (ch < '1' || ch > '9') {
					valid = false
					break
				}
			}

			if !valid {
				fmt.Println("Use only digits 1-9, 0, or .")
				continue
			}

			line = strings.ReplaceAll(line, "0", ".")
			rows = append(rows, line)
			break
		}
	}

	puzzleString := strings.Join(rows, "")

	return solver.Puzzle{
		Id:         "manual",
		Puzzle:     puzzleString,
		Difficulty: "custom",
	}, true
}

func runBenchmark(puzzles []solver.Puzzle) {
	start := time.Now()
	attempted, solved := 0, 0

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("\nBenchmark started...")

	for i, puzzle := range puzzles {
		if len(puzzle.Puzzle) != 81 {
			fmt.Printf(
				"Invalid puzzle: index=%d ID=%q length=%d puzzle=%q\n",
				i, puzzle.Id, len(puzzle.Puzzle), puzzle.Puzzle,
			)
			continue
		}
		s := &solver.Sudoku{}
		s.InitializeSudoku(puzzle)

		// Important: Solve() currently prints steps and waits for Enter.
		// Replace this with your non-interactive solving method.
		s.SolveAutomatically()

		attempted++
		if isComplete(s) {
			solved++
		}

		select {
		case <-ticker.C:
			fmt.Printf(
				"\rAttempted: %d | Solved: %d | Elapsed: %s",
				attempted,
				solved,
				time.Since(start).Round(time.Second),
			)
		default:
		}
	}

	fmt.Printf("\nBenchmark complete\n")
	fmt.Printf("Attempted: %d\n", attempted)
	fmt.Printf("Solved: %d\n", solved)
	fmt.Printf("Unsolved: %d\n", attempted-solved)
	fmt.Printf("Elapsed: %s\n", time.Since(start).Round(time.Millisecond))
}

func isComplete(s *solver.Sudoku) bool {
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			if s.Board[row][col] == 0 {
				return false
			}
		}
	}
	return true
}

func appendPuzzle(puzzles *[]solver.Puzzle, dataLine string) {
	dataArray := strings.Split(dataLine, ",")

	if len(dataLine) < 5 {
		return
	}

	puzzleItem := solver.Puzzle{
		Id:         dataArray[0],
		Puzzle:     dataArray[1],
		Solution:   dataArray[2],
		Clues:      dataArray[3],
		Difficulty: dataArray[4],
	}
	*puzzles = append(*puzzles, puzzleItem)
}
