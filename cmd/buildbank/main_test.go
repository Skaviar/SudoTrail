package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/s-threepio/godoku/internal/puzzlebank"
	"github.com/s-threepio/godoku/solver"
)

func TestBuildBankPreservesRatingsAndIncludesRealTechniques(t *testing.T) {
	data, err := os.ReadFile("../../solver/testdata/rated_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct{ ID, Puzzle, Solution, Difficulty string }
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "rated.csv")
	file, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	writer := csv.NewWriter(file)
	writer.Write([]string{"id", "puzzle", "solution", "difficulty", "rating_status", "rating_version"})
	for _, f := range fixtures {
		writer.Write([]string{f.ID, f.Puzzle, f.Solution, f.Difficulty, "solved", solver.RatingVersion})
	}
	writer.Flush()
	file.Close()
	output := filepath.Join(dir, "puzzles.json")
	if err := buildBank(input, output, 1, 17); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(output)
	var bank []puzzlebank.Puzzle
	if err := json.Unmarshal(first, &bank); err != nil || len(bank) != 4 {
		t.Fatal("missing balanced output")
	}
	for _, p := range bank {
		s, _ := solver.NewSudoku(solver.Puzzle{Puzzle: p.Puzzle})
		history := s.SolvePractice()
		names := map[string]bool{}
		for _, entry := range history.Entries {
			if entry.HasStep {
				names[entry.Step.Technique] = true
			}
		}
		if !s.IsSolved() || len(p.Techniques) == 0 {
			t.Fatal("invalid selected puzzle")
		}
		for _, name := range p.Techniques {
			if !names[name] || name == "Peer Elimination" {
				t.Fatal("advertised a technique absent from the practice history")
			}
		}
	}
	if err := buildBank(input, output, 1, 17); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(output)
	if string(first) != string(again) {
		t.Fatal("sampling is not reproducible")
	}
	if err := buildBank(input, output, 2, 17); err == nil {
		t.Fatal("silently generated an unbalanced catalog")
	}
	unchanged, _ := os.ReadFile(output)
	if string(first) != string(unchanged) {
		t.Fatal("failed generation replaced working bank")
	}
}
