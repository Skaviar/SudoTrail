package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/s-threepio/godoku/solver"
)

func main() {
	workers := flag.Int("workers", runtime.GOMAXPROCS(0), "number of benchmark workers")
	benchmark := flag.Bool("benchmark", false, "benchmark the database and exit without the menu")
	database := flag.String("database", "sudoku-3m.csv", "puzzle CSV path")
	limit := flag.Int("limit", 0, "benchmark only the first N puzzle records (0 = all)")
	flag.Parse()
	if err := validateBenchmarkOptions(*workers, *limit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments; use -help for available flags")
		os.Exit(2)
	}
	if *benchmark {
		if err := runBenchmark(*database, *workers, *limit, os.Stdout); err != nil {
			slog.Error("benchmark failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	// Database loading is only needed for random interactive puzzles. Benchmark
	// mode streams records separately instead of retaining the full CSV.
	var puzzles []solver.Puzzle
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

			s, err := solver.NewSudoku(puzzle)
			if err != nil {
				fmt.Println("Invalid puzzle:", err)
				continue
			}
			solveInteractively(s, reader)

		case "2":
			if len(puzzles) == 0 {
				puzzles, err = loadPuzzles(*database)
				if err != nil {
					fmt.Println("Cannot load database:", err)
					continue
				}
			}
			puzzle := puzzles[rng.Intn(len(puzzles))]
			fmt.Printf("\nPuzzle ID: %s | Difficulty: %s\n",
				puzzle.Id, puzzle.Difficulty)

			s, err := solver.NewSudoku(puzzle)
			if err != nil {
				fmt.Println("Invalid puzzle:", err)
				continue
			}
			solveInteractively(s, reader)

		case "3":
			if err := runBenchmark(*database, *workers, *limit, os.Stdout); err != nil {
				fmt.Println("Benchmark failed:", err)
			}

		case "0":
			return

		default:
			fmt.Println("Invalid option. Choose 0, 1, 2, or 3.")
		}
	}
}

func loadPuzzles(path string) ([]solver.Puzzle, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var puzzles []solver.Puzzle
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		appendPuzzle(&puzzles, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(puzzles) == 0 {
		return nil, fmt.Errorf("no puzzles found in %s", path)
	}
	return puzzles, nil
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

func percentage(numerator, denominator int64) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) * 100 / float64(denominator)
}

func progressBar(percent float64, width int) string {
	filled := int(percent * float64(width) / 100)
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) +
		strings.Repeat("░", width-filled) + "]"
}

func formatDuration(d time.Duration) string {
	seconds := int64(d.Seconds())
	if seconds < 0 {
		seconds = 0
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

func appendPuzzle(puzzles *[]solver.Puzzle, dataLine string) {
	dataArray := strings.Split(dataLine, ",")

	if len(dataArray) != 5 || dataArray[0] == "id" {
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
