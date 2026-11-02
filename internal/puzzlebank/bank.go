// Package puzzlebank contains the balanced practice catalog shipped to browsers.
package puzzlebank

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/s-threepio/godoku/solver"
)

type Puzzle struct {
	ID         string   `json:"id"`
	Puzzle     string   `json:"puzzle"`
	Difficulty string   `json:"difficulty"`
	Techniques []string `json:"techniques"`
}

type Selection struct {
	Difficulty string
	Technique  string
	Exclude    string
}

type Technique struct {
	Name       string            `json:"name"`
	Count      int               `json:"count"`
	Difficulty solver.Difficulty `json:"difficulty"`
}
type Level struct {
	Difficulty string      `json:"difficulty"`
	Count      int         `json:"count"`
	Techniques []Technique `json:"techniques"`
}

var Levels = [...]string{"easy", "medium", "hard", "expert"}

//go:embed puzzles.json
var catalog []byte
var puzzles = load()

func load() []Puzzle {
	var result []Puzzle
	if err := json.Unmarshal(catalog, &result); err != nil {
		panic(err)
	}
	return result
}
func Random() Puzzle { return puzzles[rand.IntN(len(puzzles))] }

func Select(selection Selection) (Puzzle, error) {
	if selection.Difficulty != "" {
		valid := false
		for _, level := range Levels {
			valid = valid || level == selection.Difficulty
		}
		if !valid {
			return Puzzle{}, fmt.Errorf("choose easy, medium, hard, or expert")
		}
	}
	var eligible []int
	for i, p := range puzzles {
		if selection.Difficulty != "" && p.Difficulty != selection.Difficulty {
			continue
		}
		matches := selection.Technique == ""
		for _, technique := range p.Techniques {
			matches = matches || technique == selection.Technique
		}
		if matches {
			eligible = append(eligible, i)
		}
	}
	if len(eligible) == 0 {
		return Puzzle{}, fmt.Errorf("no practice puzzle matches this difficulty and technique")
	}
	if len(eligible) > 1 && selection.Exclude != "" {
		filtered := eligible[:0]
		for _, i := range eligible {
			if puzzles[i].ID != selection.Exclude {
				filtered = append(filtered, i)
			}
		}
		eligible = filtered
	}
	return puzzles[eligible[rand.IntN(len(eligible))]], nil
}

func Catalog() []Level {
	levels := make([]Level, 0, len(Levels))
	for _, name := range Levels {
		level := Level{Difficulty: name, Techniques: []Technique{}}
		counts := map[string]int{}
		for _, p := range puzzles {
			if p.Difficulty == name {
				level.Count++
				for _, t := range p.Techniques {
					counts[t]++
				}
			}
		}
		for name, count := range counts {
			difficulty, _, _ := solver.TechniqueDifficulty(name)
			level.Techniques = append(level.Techniques, Technique{Name: name, Count: count, Difficulty: difficulty})
		}
		sort.Slice(level.Techniques, func(i, j int) bool { return level.Techniques[i].Name < level.Techniques[j].Name })
		levels = append(levels, level)
	}
	return levels
}
