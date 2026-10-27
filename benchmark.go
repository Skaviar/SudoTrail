package main

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/s-threepio/godoku/solver"
)

type benchmarkStats struct {
	Processed int64
	Solved    int64
	Unsolved  int64
	Invalid   int64
}

type benchmarkOutcome int

const (
	benchmarkUnsolved benchmarkOutcome = iota
	benchmarkSolved
	benchmarkInvalid
)

func validateBenchmarkOptions(workers, limit int) error {
	if workers < 1 {
		return fmt.Errorf("-workers must be at least 1")
	}
	if limit < 0 {
		return fmt.Errorf("-limit must be 0 (all puzzles) or positive")
	}
	return nil
}

// scanBenchmarkCSV retains only one CSV record at a time. Malformed record
// shapes become invalid puzzles, so they are counted rather than disappearing
// from the denominator. CSV syntax and I/O errors stop the run.
func scanBenchmarkCSV(ctx context.Context, input io.Reader, limit int, visit func(solver.Puzzle) error) (int64, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	var count int64
	for limit == 0 || count < int64(limit) {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		row, err := reader.Read()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return count, fmt.Errorf("read puzzle CSV: %w", err)
		}
		if len(row) >= 2 && strings.TrimPrefix(row[0], "\ufeff") == "id" && row[1] == "puzzle" {
			continue
		}
		var puzzle solver.Puzzle
		if len(row) == 5 {
			puzzle = solver.Puzzle{Id: row[0], Puzzle: row[1], Solution: row[2], Clues: row[3], Difficulty: row[4]}
		}
		if visit != nil {
			if err := visit(puzzle); err != nil {
				return count, err
			}
		}
		count++
	}
	return count, nil
}

// benchmarkCSV uses a bounded job queue. Each worker owns its Sudoku; the
// calling goroutine alone updates statistics and prints progress.
func benchmarkCSV(ctx context.Context, input io.Reader, workers, limit int, progress func(benchmarkStats)) (benchmarkStats, error) {
	var stats benchmarkStats
	if err := validateBenchmarkOptions(workers, limit); err != nil {
		return stats, err
	}
	jobs := make(chan solver.Puzzle, workers)
	results := make(chan benchmarkOutcome, workers)
	readerDone := make(chan error, 1)
	go func() {
		defer close(jobs)
		_, err := scanBenchmarkCSV(ctx, input, limit, func(puzzle solver.Puzzle) error {
			select {
			case jobs <- puzzle:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		readerDone <- err
	}()

	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for puzzle := range jobs {
				if ctx.Err() != nil {
					return
				}
				outcome := benchmarkUnsolved
				s, err := solver.NewSudoku(puzzle)
				if err != nil {
					outcome = benchmarkInvalid
				} else {
					s.SolveAutomatically()
					if s.IsSolved() {
						outcome = benchmarkSolved
					}
				}
				select {
				case results <- outcome:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		group.Wait()
		close(results)
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case result, ok := <-results:
			if !ok {
				err := <-readerDone
				if err == nil {
					err = ctx.Err()
				}
				return stats, err
			}
			stats.Processed++
			switch result {
			case benchmarkSolved:
				stats.Solved++
			case benchmarkUnsolved:
				stats.Unsolved++
			case benchmarkInvalid:
				stats.Invalid++
			}
		case <-ticker.C:
			if progress != nil {
				progress(stats)
			}
		}
	}
}

func runBenchmark(path string, workers, limit int, output io.Writer) error {
	if err := validateBenchmarkOptions(workers, limit); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	fmt.Fprintf(output, "\nGodoku — Database Benchmark\nDatabase: %s\nWorkers: %d\nCounting puzzle records...\n", path, workers)
	// A counting pass gives an exact denominator and ETA without retaining rows.
	total, err := scanBenchmarkCSV(ctx, file, limit, nil)
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(output, "Benchmark stopped while counting.")
		return nil
	}
	if err != nil {
		return err
	}
	if total == 0 {
		return fmt.Errorf("database contains no puzzle records")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	fmt.Fprintf(output, "Selected puzzles: %d\nPress Ctrl+C to stop and show a partial summary.\n", total)
	start := time.Now()
	stats, err := benchmarkCSV(ctx, file, workers, limit, func(stats benchmarkStats) {
		printBenchmarkProgress(output, stats, total, workers, start)
	})
	printBenchmarkSummary(output, stats, total, workers, start, err == nil)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func printBenchmarkProgress(output io.Writer, stats benchmarkStats, total int64, workers int, start time.Time) {
	elapsed := time.Since(start)
	progress := percentage(stats.Processed, total)
	rate := float64(stats.Processed) / max(elapsed.Seconds(), 0.001)
	eta := time.Duration(0)
	if rate > 0 {
		eta = time.Duration(float64(total-stats.Processed) / rate * float64(time.Second))
	}
	fmt.Fprintf(output, "Progress %s %6.2f%% (%d / %d) | Solved %d (%.2f%%) | Unsolved %d | Invalid %d | Workers %d | %.1f puzzles/sec | Elapsed %s | ETA %s\n",
		progressBar(progress, 24), progress, stats.Processed, total,
		stats.Solved, percentage(stats.Solved, stats.Processed), stats.Unsolved, stats.Invalid,
		workers, rate, formatDuration(elapsed), formatDuration(eta))
}

func printBenchmarkSummary(output io.Writer, stats benchmarkStats, total int64, workers int, start time.Time, complete bool) {
	label := "Benchmark complete"
	if !complete {
		label = "Benchmark stopped — partial results"
	}
	elapsed := time.Since(start)
	fmt.Fprintf(output, "\n%s\n%s\n", label, strings.Repeat("─", 40))
	fmt.Fprintf(output, "Total puzzles : %d\nWorkers       : %d\nProcessed     : %d\nSolved        : %d\nUnsolved      : %d\nInvalid       : %d\n",
		total, workers, stats.Processed, stats.Solved, stats.Unsolved, stats.Invalid)
	fmt.Fprintf(output, "Solve rate    : %.2f%% of processed puzzles\n", percentage(stats.Solved, stats.Processed))
	fmt.Fprintf(output, "Elapsed       : %s\nThroughput    : %.2f puzzles/sec\n",
		formatDuration(elapsed), float64(stats.Processed)/max(elapsed.Seconds(), 0.001))
}
