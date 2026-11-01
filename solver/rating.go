package solver

import (
	"context"
	"fmt"
	"strings"
)

// RatingVersion identifies both the technique weights and the rating order.
// Change it when either changes, so exported catalogs remain interpretable.
const RatingVersion = "godoku-v1"

type Difficulty string

const (
	Easy   Difficulty = "easy"
	Medium Difficulty = "medium"
	Hard   Difficulty = "hard"
	Expert Difficulty = "expert"
)

// A category describes the hardest deduction used by the rating solve path.
// Score measures accumulated work; it does not promote the category by itself.
type Rating struct {
	Version          string         `json:"version"`
	Status           string         `json:"status"` // solved, unsolved, or invalid
	Difficulty       Difficulty     `json:"difficulty"`
	Score            int            `json:"score"`
	HardestTechnique string         `json:"hardestTechnique"`
	LogicalSteps     int            `json:"logicalSteps"`
	TechniqueCounts  map[string]int `json:"techniqueCounts"`
	Error            string         `json:"error,omitempty"`
}

type techniqueRating struct {
	level  Difficulty
	points int
}

// These are Godoku weights, not HoDoKu or Sudoku Explainer scores. Keep the
// policy explicit: a new unclassified technique must fail rather than silently
// assign an incorrect category to millions of exported puzzles.
var ratingRules = map[string]techniqueRating{
	"Full House": {Easy, 1}, "Naked Single": {Easy, 1}, "Hidden Single": {Easy, 2},
	"Locked Candidates — Pointing": {Medium, 6}, "Locked Candidates — Claiming": {Medium, 6},
	"Naked Pair": {Medium, 10}, "Hidden Pair": {Medium, 12},
	"Naked Triple": {Medium, 18}, "Hidden Triple": {Medium, 20},
	"Naked Quadruple": {Hard, 30}, "Hidden Quadruple": {Hard, 35},
	"X-Wing": {Hard, 40}, "Skyscraper": {Hard, 45}, "2-String Kite": {Hard, 45},
	"Turbot Fish": {Hard, 50}, "Empty Rectangle": {Hard, 55},
	"XY-Wing": {Hard, 60}, "XYZ-Wing": {Hard, 65}, "W-Wing": {Hard, 70},
	"Swordfish": {Hard, 80}, "Jellyfish": {Hard, 100},
	"Remote Pairs": {Hard, 65}, "Simple Coloring — Color Wrap": {Hard, 70}, "Simple Coloring — Color Trap": {Hard, 70},
	"Unique Rectangle Type 1": {Hard, 70}, "Unique Rectangle Type 2": {Hard, 80}, "BUG+1": {Hard, 90},
	"Unique Rectangle Type 3": {Expert, 120}, "Unique Rectangle Type 4": {Expert, 120},
	"Unique Rectangle Type 5": {Expert, 130}, "Unique Rectangle Type 6": {Expert, 150},
	"Hidden Rectangle": {Expert, 150}, "Avoidable Rectangle Type 1": {Expert, 130}, "Avoidable Rectangle Type 2": {Expert, 130},
	"Multi-Coloring": {Expert, 140}, "3D Medusa": {Expert, 180},
	"X-Chain": {Expert, 160}, "XY-Chain": {Expert, 180}, "Alternating Inference Chain": {Expert, 220},
	"ALS-XZ": {Expert, 200}, "Doubly Linked ALS-XZ": {Expert, 240}, "ALS-XY-Wing": {Expert, 260},
	"ALS Chain": {Expert, 300}, "Death Blossom": {Expert, 320},
	"Forcing Chain": {Expert, 500}, "Forcing Net": {Expert, 800},
}

// TechniqueDifficulty also recognizes generated fish/rectangle names. False
// means the technique needs an explicit rating policy before export can proceed.
func TechniqueDifficulty(name string) (Difficulty, int, bool) {
	if name == "Peer Elimination" {
		return Easy, 0, true
	}
	if rule, ok := ratingRules[name]; ok {
		return rule.level, rule.points, true
	}
	if base, ok := strings.CutSuffix(name, " (Missing Candidates)"); ok {
		if rule, exists := ratingRules[base]; exists && (strings.HasPrefix(base, "Unique Rectangle Type ") || base == "Hidden Rectangle") {
			return Expert, rule.points + 30, true
		}
		return "", 0, false
	}
	// Fish names may combine Siamese, Finned/Sashimi, and Franken/Mutant.
	remaining := name
	bonus := 0
	for _, prefix := range []struct {
		name   string
		points int
	}{{"Siamese ", 80}, {"Finned ", 60}, {"Sashimi ", 70}, {"Franken ", 60}, {"Mutant ", 80}} {
		if rest, ok := strings.CutPrefix(remaining, prefix.name); ok {
			remaining = rest
			bonus += prefix.points
		}
	}
	for size := 2; size <= 7; size++ {
		if remaining == fishNames[size] && (bonus > 0 || size >= 5) {
			return Expert, 60 + 20*size + bonus, true
		}
	}
	return "", 0, false
}

func difficultyRank(d Difficulty) int {
	switch d {
	case Easy:
		return 0
	case Medium:
		return 1
	case Hard:
		return 2
	case Expert:
		return 3
	}
	return -1
}

func newRating() Rating {
	return Rating{Version: RatingVersion, Status: "unsolved", TechniqueCounts: map[string]int{}}
}

// RatePuzzle always starts from the supplied clues and never reads Solution or
// the source Difficulty. Invalid local clues get an explicit ungraded record.
func RatePuzzle(ctx context.Context, p Puzzle) (Rating, error) {
	rating := newRating()
	if err := ctx.Err(); err != nil {
		return rating, err
	}
	s, err := NewSudoku(p)
	if err != nil {
		rating.Status = "invalid"
		rating.Error = err.Error()
		return rating, nil
	}
	return s.SolveRated(ctx)
}

// SolveRated uses immediate peer cleanup, so that playback-only cleanup steps
// do not inflate scores or hint counts. It does not allocate board histories.
// After every deduction, all easier tiers get another chance before Expert.
func (s *Sudoku) SolveRated(ctx context.Context) (Rating, error) {
	rating := newRating()
	hardest := Easy
	hardestPoints := 0
	if err := ctx.Err(); err != nil {
		return rating, err
	}
	s.applyPendingCandidates()
	for {
		if err := ctx.Err(); err != nil {
			return rating, err
		}
		step, ok := s.applyRatingTechnique()
		if !ok {
			if s.IsSolved() {
				rating.Status = "solved"
				rating.Difficulty = hardest
			}
			return rating, nil
		}
		if step.Technique == "Peer Elimination" {
			continue
		}
		level, points, known := TechniqueDifficulty(step.Technique)
		if !known {
			return rating, fmt.Errorf("no rating rule for technique %q", step.Technique)
		}
		rating.LogicalSteps++
		rating.Score += points
		rating.TechniqueCounts[step.Technique]++
		if difficultyRank(level) > difficultyRank(hardest) || level == hardest && points > hardestPoints {
			hardest, hardestPoints = level, points
			rating.HardestTechnique = step.Technique
		}
	}
}

func (s *Sudoku) applyRatingTechnique() (Step, bool) {
	// Easy: singles only.
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyFullHouse, ApplyHiddenSingle, ApplyNakedSingle} {
		if step, ok := apply(s); ok {
			return step, true
		}
	}
	// Medium: intersections and pairs/triples.
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyPointing, ApplyClaiming} {
		if step, ok := apply(s); ok {
			return step, true
		}
	}
	for size := 2; size <= 3; size++ {
		if step, ok := ApplyNakedSubset(s, size); ok {
			return step, true
		}
		if step, ok := ApplyHiddenSubset(s, size); ok {
			return step, true
		}
	}
	// Hard: keep these ahead of finned fish and the advanced scanner, even
	// though the teaching solver's established order is slightly different.
	if step, ok := ApplyNakedSubset(s, 4); ok {
		return step, true
	}
	if step, ok := ApplyHiddenSubset(s, 4); ok {
		return step, true
	}
	if step, ok := ApplyBasicFish(s, 2); ok {
		return step, true
	}
	for _, apply := range []func(*Sudoku) (Step, bool){ApplySkyscraper, ApplyTwoStringKite, ApplyTurbotFish, ApplyEmptyRectangle, ApplyXYWing, ApplyXYZWing, ApplyWWing} {
		if step, ok := apply(s); ok {
			return step, true
		}
	}
	for size := 3; size <= 4; size++ {
		if step, ok := ApplyBasicFish(s, size); ok {
			return step, true
		}
	}
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyRemotePairs, ApplySimpleColoring} {
		if step, ok := apply(s); ok {
			return step, true
		}
	}
	for kind := 1; kind <= 2; kind++ {
		if step, ok := ApplyUniqueRectangle(s, kind); ok {
			return step, true
		}
	}
	if step, ok := ApplyBUGPlusOne(s); ok {
		return step, true
	}
	// Expert: every remaining logical family, including the shared chain,
	// ALS, and forcing implementations. Backtracking stays uniqueness-only.
	for size := 5; size <= 7; size++ {
		if step, ok := ApplyBasicFish(s, size); ok {
			return step, true
		}
	}
	for size := 2; size <= 7; size++ {
		if step, ok := ApplyFinnedFish(s, size); ok {
			return step, true
		}
	}
	return s.applyAdvancedTechniques()
}
