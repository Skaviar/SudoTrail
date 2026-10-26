package puzzleapp

import (
	"strings"
	"testing"
)

func TestSharedSolverContract(t *testing.T) {
	puzzle := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	if !Validate(puzzle).Valid {
		t.Fatal("valid puzzle rejected")
	}
	response, err := Solve(puzzle)
	if err != nil || !response.Solved || response.Status != "solved" {
		t.Fatalf("unexpected result: %v, %d states", err, len(response.States))
	}
	placements, cleanups := 0, 0
	for i, state := range response.States[1:] {
		if state.Step.Kind == "placement" {
			placements++
		}
		if state.Step.Technique == "Peer Elimination" {
			cleanups++
			if state.Board != response.States[i].Board {
				t.Fatal("peer cleanup changed board values")
			}
		}
	}
	if placements != 51 || cleanups == 0 {
		t.Fatalf("expected 51 placements and separate cleanup steps, got %d/%d", placements, cleanups)
	}
	if response.States[0].Step != nil || response.States[0].Board[0][2] != 0 {
		t.Fatal("lost initial state")
	}
	if response.States[1].Step.Visual == nil {
		t.Fatal("lost visual reasoning")
	}
	if _, err := Solve("11" + strings.Repeat(".", 79)); err == nil {
		t.Fatal("invalid puzzle accepted")
	}
	stuck, err := Solve(strings.Repeat(".", 81))
	if err != nil || stuck.Solved || stuck.Status != "stuck" {
		t.Fatal("stuck status incorrect")
	}
}
