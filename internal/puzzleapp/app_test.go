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

func TestHintGuidesWithoutRevealingAnswer(t *testing.T) {
	puzzle := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	hint, err := Hint(puzzle)
	if err != nil || hint.Status != "hint" || hint.Focus == nil || hint.Guidance == "" || hint.Technique == "" {
		t.Fatalf("expected guidance: %+v, %v", hint, err)
	}
	if puzzle[hint.Focus.Row*9+hint.Focus.Col] != '0' {
		t.Fatal("hint focused on a given")
	}
	for _, tc := range []struct{ puzzle, status string }{
		{strings.Repeat(".", 81), "stuck"},
		{"534678912672195348198342567859761423426853791713924856961537284287419635345286179", "solved"},
	} {
		hint, err := Hint(tc.puzzle)
		if err != nil || hint.Status != tc.status || hint.Guidance != "" {
			t.Fatalf("unexpected terminal hint: %+v, %v", hint, err)
		}
	}
	if _, err := Hint("11" + strings.Repeat(".", 79)); err == nil {
		t.Fatal("invalid puzzle accepted")
	}
}
