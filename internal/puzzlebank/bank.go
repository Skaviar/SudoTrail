// Package puzzlebank contains the small random-puzzle catalog shipped to browsers.
package puzzlebank

import (
	_ "embed"
	"encoding/json"
	"math/rand/v2"
)

type Puzzle struct {
	ID         string `json:"id"`
	Puzzle     string `json:"puzzle"`
	Difficulty string `json:"difficulty"`
}

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
