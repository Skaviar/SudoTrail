package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/s-threepio/godoku/solver"
)

const benchmarkSolvedGrid = "534678912672195348198342567859761423426853791713924856961537284287419635345286179"
const benchmarkEasyPuzzle = "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
const benchmarkHeader = "id,puzzle,solution,clues,difficulty\n"

func benchmarkRecord(id int, puzzle string) string {
	return fmt.Sprintf("%d,%s,%s,30,1.0\n", id, puzzle, benchmarkSolvedGrid)
}

func TestBenchmarkWorkersAgree(t *testing.T) {
	var input strings.Builder
	input.WriteString(benchmarkHeader)
	// Include solved, solvable, stuck, invalid-clue, and malformed CSV records.
	for i := range 12 {
		input.WriteString(benchmarkRecord(i*5, benchmarkSolvedGrid))
		input.WriteString(benchmarkRecord(i*5+1, benchmarkEasyPuzzle))
		input.WriteString(benchmarkRecord(i*5+2, strings.Repeat(".", 81)))
		input.WriteString(benchmarkRecord(i*5+3, "11"+strings.Repeat(".", 79)))
		input.WriteString("missing,fields\n")
	}
	want := benchmarkStats{Processed: 60, Solved: 24, Unsolved: 12, Invalid: 24}
	for _, workers := range []int{1, 2, 8, 64} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			got, err := benchmarkCSV(context.Background(), strings.NewReader(input.String()), workers, 0, nil)
			if err != nil || got != want {
				t.Fatalf("got %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func TestBenchmarkLimitExcludesHeaderAndStopsBeforeBadTail(t *testing.T) {
	input := benchmarkHeader + benchmarkRecord(1, benchmarkSolvedGrid) + benchmarkRecord(2, strings.Repeat(".", 81)) + "broken,\"unterminated\n"
	for _, workers := range []int{1, 8} {
		got, err := benchmarkCSV(context.Background(), strings.NewReader(input), workers, 2, nil)
		want := benchmarkStats{Processed: 2, Solved: 1, Unsolved: 1}
		if err != nil || got != want {
			t.Fatalf("limit did not stop after two puzzles: %+v, %v", got, err)
		}
	}
}

func TestBenchmarkCSVQuotedFieldsAndWindowsLines(t *testing.T) {
	input := "\ufeffid,puzzle,solution,clues,difficulty\r\n" + fmt.Sprintf("\"id,with,commas\",%s,%s,81,\"easy,complete\"\r\n", benchmarkSolvedGrid, benchmarkSolvedGrid)
	var received []solver.Puzzle
	count, err := scanBenchmarkCSV(context.Background(), strings.NewReader(input), 0, func(p solver.Puzzle) error {
		received = append(received, p)
		return nil
	})
	if err != nil || count != 1 || len(received) != 1 || received[0].Id != "id,with,commas" || received[0].Difficulty != "easy,complete" {
		t.Fatalf("CSV fields were not decoded correctly: %+v, %d, %v", received, count, err)
	}
}

func TestBenchmarkReturnsCSVErrorsAndCountsFinishedJobs(t *testing.T) {
	input := benchmarkHeader + benchmarkRecord(1, benchmarkSolvedGrid) + "broken,\"unterminated\n"
	got, err := benchmarkCSV(context.Background(), strings.NewReader(input), 4, 0, nil)
	if err == nil || got != (benchmarkStats{Processed: 1, Solved: 1}) {
		t.Fatalf("CSV error lost or completed job not counted: %+v, %v", got, err)
	}
	wantErr := errors.New("read failed")
	if _, err := benchmarkCSV(context.Background(), iotest.ErrReader(wantErr), 2, 0, nil); !errors.Is(err, wantErr) {
		t.Fatalf("I/O error not preserved: %v", err)
	}
}

func TestBenchmarkOptionsAndCancellation(t *testing.T) {
	for _, tt := range []struct{ workers, limit int }{{0, 0}, {-1, 0}, {1, -1}} {
		if _, err := benchmarkCSV(context.Background(), strings.NewReader(""), tt.workers, tt.limit, nil); err == nil {
			t.Fatal("accepted invalid worker count or limit")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stats, err := benchmarkCSV(ctx, strings.NewReader(benchmarkRecord(1, benchmarkSolvedGrid)), 8, 0, nil)
	if !errors.Is(err, context.Canceled) || stats != (benchmarkStats{}) {
		t.Fatalf("canceled run processed jobs or lost cancellation: %+v, %v", stats, err)
	}
}

func TestBenchmarkSummaryUsesAllProcessedRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.csv")
	input := benchmarkHeader + benchmarkRecord(1, benchmarkSolvedGrid) + benchmarkRecord(2, benchmarkEasyPuzzle) + benchmarkRecord(3, strings.Repeat(".", 81)) + "bad,row\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runBenchmark(path, 3, 0, &output); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Benchmark complete", "Total puzzles : 4", "Workers       : 3", "Processed     : 4", "Solved        : 2", "Unsolved      : 1", "Invalid       : 1", "Solve rate    : 50.00% of processed puzzles"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("summary missing %q:\n%s", text, output.String())
		}
	}
}

func TestBenchmarkEmptyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(path, []byte(benchmarkHeader), 0600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := runBenchmark(path, 2, 0, &output); err == nil || strings.Contains(output.String(), "Benchmark complete") {
		t.Fatal("empty database reported a completed benchmark")
	}
}
