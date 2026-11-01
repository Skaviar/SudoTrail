package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/s-threepio/godoku/solver"
)

// Keep the first five columns compatible with the terminal and HTTP catalogs.
var ratingCSVHeader = []string{
	"id", "puzzle", "solution", "clues", "difficulty", "original_difficulty",
	"score", "hardest_technique", "logical_steps", "technique_counts",
	"rating_status", "rating_version", "rating_error",
}

var ratingLevels = [...]solver.Difficulty{solver.Easy, solver.Medium, solver.Hard, solver.Expert}

type ratingStats struct {
	Processed int64
	Counts    [4]int64
	Unsolved  int64
	Invalid   int64
}

func (s *ratingStats) record(r solver.Rating) {
	s.Processed++
	switch r.Status {
	case "solved":
		for i, level := range ratingLevels {
			if r.Difficulty == level {
				s.Counts[i]++
				return
			}
		}
	case "unsolved":
		s.Unsolved++
	case "invalid":
		s.Invalid++
	}
}

type ratingJob struct {
	sequence int64
	puzzle   solver.Puzzle
	invalid  string
}
type ratingResult struct {
	job    ratingJob
	rating solver.Rating
	err    error
}

// The reader accepts the original five-column catalog and the exported format.
// Malformed record shapes are preserved as ungraded invalid rows. CSV syntax
// errors stop the stream; previously written rows remain a valid CSV prefix.
func readRatingJobs(ctx context.Context, input io.Reader, limit int, visit func(ratingJob) error) error {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	originalColumn := 4
	for sequence := int64(0); limit == 0 || sequence < int64(limit); {
		if err := ctx.Err(); err != nil {
			return err
		}
		row, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read puzzle CSV: %w", err)
		}
		if len(row) >= 2 && strings.TrimPrefix(row[0], "\ufeff") == "id" && row[1] == "puzzle" {
			for i, name := range row {
				if name == "original_difficulty" {
					originalColumn = i
				}
			}
			continue
		}
		field := func(i int) string {
			if i < len(row) {
				return row[i]
			}
			return ""
		}
		job := ratingJob{sequence: sequence, puzzle: solver.Puzzle{Id: field(0), Puzzle: field(1), Solution: field(2), Clues: field(3), Difficulty: field(originalColumn)}}
		if len(row) < 5 {
			job.invalid = "expected at least five puzzle columns"
		}
		if err := visit(job); err != nil {
			return err
		}
		sequence++
	}
	return nil
}

func ratingRecord(result ratingResult) []string {
	p, r := result.job.puzzle, result.rating
	counts, _ := json.Marshal(r.TechniqueCounts) // map[string]int always encodes.
	return []string{p.Id, p.Puzzle, p.Solution, p.Clues, string(r.Difficulty), p.Difficulty,
		strconv.Itoa(r.Score), r.HardestTechnique, strconv.Itoa(r.LogicalSteps), string(counts), r.Status, r.Version, r.Error}
}

// rateCSV preserves input order with a bounded reorder window. A credit is
// returned only when its row is written, so one slow early puzzle cannot make
// millions of later completed results accumulate in memory.
func rateCSV(ctx context.Context, input io.Reader, output io.Writer, workers, limit int, progress func(ratingStats)) (ratingStats, error) {
	var stats ratingStats
	if err := validateBenchmarkOptions(workers, limit); err != nil {
		return stats, err
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	writer := csv.NewWriter(output)
	if err := writer.Write(ratingCSVHeader); err != nil {
		return stats, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return stats, err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan ratingJob, workers)
	results := make(chan ratingResult, workers)
	credits := make(chan struct{}, workers*2)
	readerDone := make(chan error, 1)
	go func() {
		defer close(jobs)
		readerDone <- readRatingJobs(ctx, input, limit, func(job ratingJob) error {
			select {
			case credits <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case jobs <- job:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				result := ratingResult{job: job}
				if job.invalid != "" {
					result.rating = solver.Rating{Version: solver.RatingVersion, Status: "invalid", Error: job.invalid, TechniqueCounts: map[string]int{}}
				} else {
					result.rating, result.err = solver.RatePuzzle(ctx, job.puzzle)
				}
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() { group.Wait(); close(results) }()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	pending := make(map[int64]ratingResult)
	next := int64(0)
	var outputErr error
	for results != nil {
		select {
		case result, ok := <-results:
			if !ok {
				results = nil
				continue
			}
			if outputErr != nil || ctx.Err() != nil {
				continue
			}
			if result.err != nil {
				outputErr = result.err
				cancel()
				continue
			}
			pending[result.job.sequence] = result
			for {
				ready, exists := pending[next]
				if !exists {
					break
				}
				if err := writer.Write(ratingRecord(ready)); err != nil {
					outputErr = err
					cancel()
					break
				}
				stats.record(ready.rating)
				delete(pending, next)
				next++
				<-credits
			}
		case <-ticker.C:
			writer.Flush()
			if err := writer.Error(); err != nil && outputErr == nil {
				outputErr = err
				cancel()
			}
			if progress != nil {
				progress(stats)
			}
		}
	}
	readErr := <-readerDone // Join the producer before its input file is closed.
	writer.Flush()
	if outputErr != nil {
		return stats, outputErr
	}
	if err := writer.Error(); err != nil {
		return stats, err
	}
	if readErr != nil {
		return stats, readErr
	}
	return stats, ctx.Err()
}

func runRatingExport(path, destination string, workers, limit int, output io.Writer) error {
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
	fmt.Fprintf(output, "\nGodoku — Difficulty Export\nDatabase: %s\nOutput: %s\nWorkers: %d\nRating model: %s\nCounting puzzle records...\n", path, destination, workers, solver.RatingVersion)
	total, err := scanBenchmarkCSV(ctx, file, limit, nil)
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(output, "Rating stopped while counting; no output created.")
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
	// Never truncate a source catalog or a previous export, including aliases
	// through symlinks/hardlinks. The caller can choose a different output path.
	resultFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("create new rating file (choose an unused -output path): %w", err)
	}
	fmt.Fprintf(output, "Selected puzzles: %d\nPress Ctrl+C to stop and keep a partial export.\n", total)
	start := time.Now()
	stats, rateErr := rateCSV(ctx, file, resultFile, workers, limit, func(stats ratingStats) {
		elapsed := time.Since(start)
		rate := float64(stats.Processed) / max(elapsed.Seconds(), 0.001)
		eta := time.Duration(float64(total-stats.Processed) / max(rate, 0.001) * float64(time.Second))
		fmt.Fprintf(output, "Progress %s %6.2f%% (%d / %d) | Easy %d | Medium %d | Hard %d | Expert %d | %.1f puzzles/sec | Elapsed %s | ETA %s\n", progressBar(percentage(stats.Processed, total), 24), percentage(stats.Processed, total), stats.Processed, total, stats.Counts[0], stats.Counts[1], stats.Counts[2], stats.Counts[3], rate, formatDuration(elapsed), formatDuration(eta))
	})
	if closeErr := resultFile.Close(); rateErr == nil {
		rateErr = closeErr
	}
	label := "Difficulty export complete"
	if rateErr != nil {
		label = "Difficulty export stopped — partial results"
	}
	fmt.Fprintf(output, "\n%s\n%s\nTotal puzzles : %d\nProcessed     : %d\n", label, strings.Repeat("─", 40), total, stats.Processed)
	for i, level := range ratingLevels {
		fmt.Fprintf(output, "%-13s : %d (%.2f%%)\n", level, stats.Counts[i], percentage(stats.Counts[i], stats.Processed))
	}
	fmt.Fprintf(output, "Unsolved      : %d\nInvalid       : %d\nElapsed       : %s\nOutput        : %s\n", stats.Unsolved, stats.Invalid, formatDuration(time.Since(start)), destination)
	if errors.Is(rateErr, context.Canceled) {
		return nil
	}
	return rateErr
}
