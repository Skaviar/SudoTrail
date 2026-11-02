// Build a reproducible balanced browser catalog from a rated CSV. Solutions
// are used only to verify selections and are never written into the bank.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/s-threepio/godoku/internal/puzzlebank"
	"github.com/s-threepio/godoku/solver"
)

type sample struct {
	puzzle   puzzlebank.Puzzle
	solution string
}

func main() {
	in := flag.String("in", "sudoku-rated.csv", "rated puzzle CSV")
	out := flag.String("out", "internal/puzzlebank/puzzles.json", "generated browser catalog")
	count := flag.Int("per-level", 500, "puzzles per difficulty")
	seed := flag.Int64("seed", 20261101, "deterministic reservoir sampling seed")
	flag.Parse()
	if err := buildBank(*in, *out, *count, *seed); err != nil {
		log.Fatal(err)
	}
}

func buildBank(input, output string, count int, seed int64) error {
	if count < 1 {
		return fmt.Errorf("per-level must be positive")
	}
	file, err := os.Open(input)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return err
	}
	columns := map[string]int{}
	for i, name := range header {
		columns[name] = i
	}
	for _, name := range []string{"id", "puzzle", "solution", "difficulty", "rating_status", "rating_version"} {
		if _, ok := columns[name]; !ok {
			return fmt.Errorf("missing rated CSV column %s", name)
		}
	}
	rng := rand.New(rand.NewSource(seed))
	seen := map[string]int{}
	samples := map[string][]sample{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		get := func(name string) string { return row[columns[name]] }
		level := get("difficulty")
		if get("rating_status") != "solved" {
			continue
		}
		valid := false
		for _, name := range puzzlebank.Levels {
			valid = valid || name == level
		}
		if !valid {
			return fmt.Errorf("unknown difficulty %q", level)
		}
		if get("rating_version") != solver.RatingVersion {
			return fmt.Errorf("rating model must be %s", solver.RatingVersion)
		}
		seen[level]++
		index := len(samples[level])
		if index >= count {
			index = rng.Intn(seen[level])
			if index >= count {
				continue
			}
		}
		p := sample{puzzle: puzzlebank.Puzzle{ID: get("id"), Puzzle: get("puzzle"), Difficulty: level}, solution: get("solution")}
		if index < len(samples[level]) {
			samples[level][index] = p
		} else {
			samples[level] = append(samples[level], p)
		}
	}
	var bank []puzzlebank.Puzzle
	unique := map[string]bool{}
	for _, level := range puzzlebank.Levels {
		if len(samples[level]) != count {
			return fmt.Errorf("need %d %s puzzles, found %d", count, level, len(samples[level]))
		}
		sort.Slice(samples[level], func(i, j int) bool {
			a, _ := strconv.Atoi(samples[level][i].puzzle.ID)
			b, _ := strconv.Atoi(samples[level][j].puzzle.ID)
			return a < b
		})
		for _, sample := range samples[level] {
			p := sample.puzzle
			if unique[p.Puzzle] {
				return fmt.Errorf("duplicate puzzle %s in sampled catalog; choose another seed", p.ID)
			}
			unique[p.Puzzle] = true
			s, err := solver.NewSudoku(solver.Puzzle{Puzzle: p.Puzzle})
			if err != nil {
				return fmt.Errorf("puzzle %s: %w", p.ID, err)
			}
			history := s.SolvePractice()
			if !s.IsSolved() || len(sample.solution) != 81 {
				return fmt.Errorf("practice solver cannot complete puzzle %s", p.ID)
			}
			for i, digit := range []byte(sample.solution) {
				if int8(digit-'0') != s.Board[i/9][i%9] {
					return fmt.Errorf("puzzle %s disagrees with its reference solution", p.ID)
				}
			}
			techniques := map[string]bool{}
			for _, entry := range history.Entries {
				if entry.HasStep && entry.Step.Technique != "Peer Elimination" {
					techniques[entry.Step.Technique] = true
				}
			}
			for name := range techniques {
				p.Techniques = append(p.Techniques, name)
			}
			sort.Strings(p.Techniques)
			bank = append(bank, p)
		}
		fmt.Printf("Verified %d %s puzzles and their practice techniques.\n", count, level)
	}
	encoded, err := json.Marshal(bank)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, append(encoded, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Browser bank: %d puzzles, %.0f KiB of JSON.\n", len(bank), float64(len(encoded))/1024)
	return nil
}
