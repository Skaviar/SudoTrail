package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
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

			s, err := solver.NewSudoku(puzzle)
			if err != nil {
				fmt.Println("Invalid puzzle:", err)
				continue
			}
			solveInteractively(s, reader)

		case "2":
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

type BenchmarkStats struct {
	Attempted atomic.Int64
	Solved    atomic.Int64
}

func runBenchmark(puzzles []solver.Puzzle) {
	const reportInterval = 5 * time.Second

	var stats BenchmarkStats
	start := time.Now()
	stop := make(chan struct{})
	reporterDone := make(chan struct{})

	fmt.Println("\nGodoku — Database Benchmark")
	fmt.Println(strings.Repeat("─", 64))

	go func() {
		defer close(reporterDone)
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				printBenchmarkProgress(&stats, len(puzzles), start)
			case <-stop:
				return
			}
		}
	}()

	for _, puzzle := range puzzles {
		if len(puzzle.Puzzle) != 81 {
			continue
		}
		s, err := solver.NewSudoku(puzzle)
		if err != nil {
			continue
		}
		s.SolveAutomatically()

		stats.Attempted.Add(1)
		if s.IsSolved() {
			stats.Solved.Add(1)
		}
	}

	close(stop)
	<-reporterDone

	printBenchmarkSummary(&stats, len(puzzles), start)
}

func printBenchmarkProgress(stats *BenchmarkStats, total int, start time.Time) {
	attempted := stats.Attempted.Load()
	solved := stats.Solved.Load()
	elapsed := time.Since(start)

	progress := percentage(attempted, int64(total))
	solveRate := percentage(solved, attempted)
	rate := float64(attempted) / max(elapsed.Seconds(), 0.001)

	remaining := int64(total) - attempted
	eta := time.Duration(0)
	if rate > 0 {
		eta = time.Duration(float64(remaining)/rate) * time.Second
	}

	fmt.Printf("\033[H\033[2J")
	fmt.Println("Godoku — Database Benchmark")
	fmt.Println(strings.Repeat("─", 64))
	fmt.Printf("Progress   %s %6.2f%%  (%d / %d)\n",
		progressBar(progress, 24), progress, attempted, total)
	fmt.Printf("Solved     %-10d Unsolved     %d\n",
		solved, attempted-solved)
	fmt.Printf("Solve rate %-10s Throughput   %.1f puzzles/sec\n",
		fmt.Sprintf("%.2f%%", solveRate), rate)
	fmt.Printf("Elapsed    %-10s ETA          %s\n",
		formatDuration(elapsed), formatDuration(eta))
	fmt.Println(strings.Repeat("─", 64))
}

func printBenchmarkSummary(stats *BenchmarkStats, total int, start time.Time) {
	attempted := stats.Attempted.Load()
	solved := stats.Solved.Load()
	elapsed := time.Since(start)

	fmt.Printf("\nBenchmark complete\n")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Total puzzles : %d\n", total)
	fmt.Printf("Attempted     : %d\n", attempted)
	fmt.Printf("Solved        : %d\n", solved)
	fmt.Printf("Unsolved      : %d\n", attempted-solved)
	fmt.Printf("Solve rate    : %.2f%%\n", percentage(solved, attempted))
	fmt.Printf("Elapsed       : %s\n", formatDuration(elapsed))
	fmt.Printf("Throughput    : %.2f puzzles/sec\n",
		float64(attempted)/max(elapsed.Seconds(), 0.001))
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
