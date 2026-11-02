package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/s-threepio/godoku/solver"
)

const ratingExpertPuzzle = "8..9........524.....5.1.67..2......45.17....3.......164....8..1....6...7......89."

func ratingInput(t *testing.T, rows [][]string) string {
	t.Helper()
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	writer.Write([]string{"id", "puzzle", "solution", "clues", "difficulty"})
	if err := writer.WriteAll(rows); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestRatingExportIsOrderedAndIndependentOfWorkers(t *testing.T) {
	var rows [][]string
	for i := range 3 {
		rows = append(rows,
			[]string{fmt.Sprint(i, "-expert"), ratingExpertPuzzle, "supplied answer", "24", "old rating"},
			[]string{fmt.Sprint(i, "-complete"), benchmarkSolvedGrid, benchmarkSolvedGrid, "81", "old rating"},
			[]string{fmt.Sprint(i, "-easy,\nquoted"), benchmarkEasyPuzzle, benchmarkSolvedGrid, "30", "old,quoted rating"},
			[]string{fmt.Sprint(i, "-unsolved"), strings.Repeat(".", 81), "", "0", "unknown"},
			[]string{fmt.Sprint(i, "-invalid"), "11" + strings.Repeat(".", 79), "", "2", "old rating"},
			[]string{fmt.Sprint(i, "-missing"), "fields"})
	}
	input := ratingInput(t, rows)
	var baseline string
	for _, workers := range []int{1, 2, 8} {
		var output bytes.Buffer
		stats, err := rateCSV(context.Background(), strings.NewReader(input), &output, workers, 0, nil)
		want := ratingStats{Processed: 18, Counts: [4]int64{6, 0, 0, 3}, Unsolved: 3, Invalid: 6}
		if err != nil || stats != want {
			t.Fatalf("wrong export counts: %+v %v", stats, err)
		}
		if workers == 1 {
			baseline = output.String()
		} else if output.String() != baseline {
			t.Fatal("worker count changed CSV order or ratings")
		}
		actual, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
		if err != nil || len(actual) != 19 || !reflect.DeepEqual(actual[0], ratingCSVHeader) {
			t.Fatal("export is not a valid complete CSV")
		}
		for i, row := range actual[1:] {
			if row[0] != rows[i][0] || row[1] != rows[i][1] || row[11] != solver.RatingVersion {
				t.Fatal("input metadata or order changed")
			}
			if len(rows[i]) >= 5 && row[5] != rows[i][4] {
				t.Fatal("lost original difficulty")
			}
			var counts map[string]int
			if err := json.Unmarshal([]byte(row[9]), &counts); err != nil {
				t.Fatal("invalid technique counts JSON")
			}
			if row[10] != "solved" && row[4] != "" {
				t.Fatal("assigned a solved category to an ungraded puzzle")
			}
		}
	}
	// Re-rating preserves the original difficulty and ratings. A source row
	// that originally lacked fields now has a complete CSV shape, so its
	// validation error can describe its invalid puzzle instead of its shape.
	var again bytes.Buffer
	if _, err := rateCSV(context.Background(), strings.NewReader(baseline), &again, 4, 0, nil); err != nil {
		t.Fatal("rated catalog did not round-trip")
	}
	before, _ := csv.NewReader(strings.NewReader(baseline)).ReadAll()
	after, err := csv.NewReader(&again).ReadAll()
	if err != nil || len(before) != len(after) {
		t.Fatal("re-rating lost rows")
	}
	for i := range before {
		if !reflect.DeepEqual(before[i][:12], after[i][:12]) {
			t.Fatal("re-rating changed original metadata or classification")
		}
	}
}

func TestRatingExportLimitAndMalformedTail(t *testing.T) {
	input := benchmarkHeader + benchmarkRecord(1, benchmarkSolvedGrid) + benchmarkRecord(2, benchmarkEasyPuzzle) + "broken,\"unterminated\n"
	for _, limit := range []int{0, 2} {
		var output bytes.Buffer
		stats, err := rateCSV(context.Background(), strings.NewReader(input), &output, 4, limit, nil)
		if (err == nil) != (limit == 2) || stats.Processed != 2 {
			t.Fatalf("incorrect limit/error handling: %+v %v", stats, err)
		}
		rows, parseErr := csv.NewReader(&output).ReadAll()
		if parseErr != nil || len(rows) != 3 || rows[1][0] != "1" || rows[2][0] != "2" {
			t.Fatal("lost valid prefix before CSV syntax error")
		}
	}
}

type ratingCancelWriter struct {
	bytes.Buffer
	cancel context.CancelFunc
}

func (w *ratingCancelWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if w.Len() > 4096 {
		w.cancel()
	}
	return n, err
}

type ratingFailWriter struct {
	remaining int
	err       error
}

func (w *ratingFailWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, w.err
	}
	w.remaining -= len(p)
	return len(p), nil
}

func TestRatingExportCancellationAndIOFailures(t *testing.T) {
	input := benchmarkHeader
	for i := range 500 {
		input += benchmarkRecord(i, benchmarkSolvedGrid)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := &ratingCancelWriter{cancel: cancel}
	stats, err := rateCSV(ctx, strings.NewReader(input), output, 8, 0, nil)
	if !errors.Is(err, context.Canceled) || stats.Processed <= 0 || stats.Processed >= 500 {
		t.Fatalf("did not stop active workers: %+v %v", stats, err)
	}
	rows, parseErr := csv.NewReader(bytes.NewReader(output.Bytes())).ReadAll()
	if parseErr != nil || int64(len(rows)-1) != stats.Processed {
		t.Fatal("interrupted export left an incomplete CSV row")
	}
	for i, row := range rows[1:] {
		if row[0] != fmt.Sprint(i) {
			t.Fatal("interrupted export is not an ordered prefix")
		}
	}
	wantErr := errors.New("disk full")
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if _, err := rateCSV(ctx, strings.NewReader(input), &ratingFailWriter{remaining: 8192, err: wantErr}, 8, 0, nil); !errors.Is(err, wantErr) {
		t.Fatalf("lost write failure or workers did not exit: %v", err)
	}
	var sink bytes.Buffer
	if _, err := rateCSV(context.Background(), iotest.ErrReader(wantErr), &sink, 2, 0, nil); !errors.Is(err, wantErr) {
		t.Fatal("lost source I/O error")
	}
	for _, options := range [][2]int{{0, 0}, {-1, 0}, {1, -1}} {
		if _, err := rateCSV(context.Background(), strings.NewReader(input), io.Discard, options[0], options[1], nil); err == nil {
			t.Fatal("accepted invalid export options")
		}
	}
}

func TestRatingFileExportProtectsExistingFilesAndCanBeLoaded(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.csv")
	destination := filepath.Join(dir, "rated.csv")
	input := ratingInput(t, [][]string{{"quoted,id", benchmarkEasyPuzzle, benchmarkSolvedGrid, "30", "original"}, {"done", benchmarkSolvedGrid, benchmarkSolvedGrid, "81", "original"}})
	if err := os.WriteFile(source, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	if err := runRatingExport(source, destination, 4, 0, &log); err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"Difficulty export complete", "Processed     : 2", "(100.00%)", "Unsolved      : 0", "Invalid       : 0"} {
		if !strings.Contains(log.String(), phrase) {
			t.Fatalf("missing summary: %s", phrase)
		}
	}
	data, _ := os.ReadFile(destination)
	stats, err := benchmarkCSV(context.Background(), bytes.NewReader(data), 2, 0, nil)
	if err != nil || stats.Solved != 2 || stats.Invalid != 0 {
		t.Fatal("benchmark cannot read rated catalog")
	}
	if err := runRatingExport(source, destination, 2, 0, io.Discard); !errors.Is(err, os.ErrExist) {
		t.Fatal("existing export was not protected")
	}
	again, _ := os.ReadFile(destination)
	if !bytes.Equal(data, again) {
		t.Fatal("existing export was modified")
	}
	if err := runRatingExport(source, source, 2, 0, io.Discard); !errors.Is(err, os.ErrExist) {
		t.Fatal("source file was not protected")
	}
	alias := filepath.Join(dir, "alias.csv")
	if err := os.Symlink(source, alias); err != nil {
		t.Fatal(err)
	}
	if err := runRatingExport(source, alias, 2, 0, io.Discard); !errors.Is(err, os.ErrExist) {
		t.Fatal("source alias was not protected")
	}
	original, _ := os.ReadFile(source)
	if string(original) != input {
		t.Fatal("source catalog changed")
	}
}

func TestRatingFileExportRejectsEmptyDatabase(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "empty.csv")
	destination := filepath.Join(dir, "rated.csv")
	if err := os.WriteFile(source, []byte(benchmarkHeader), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runRatingExport(source, destination, 2, 0, io.Discard); err == nil {
		t.Fatal("empty database was reported complete")
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("created an export before checking the input")
	}
}
