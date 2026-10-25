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
	if err != nil || !response.Solved || response.Status != "solved" || len(response.States) != 52 {
		t.Fatalf("unexpected result: %v, %d states", err, len(response.States))
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
