package webapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/s-threepio/godoku/solver"
)

func TestJSONContract(t *testing.T) {
	h := NewHandler(nil)
	for _, path := range []string{"/api/validate", "/api/solve"} {
		for _, body := range []string{`null`, `{}`, `{"puzzle":null}`, `[]`, `"text"`, `{"puzzle":true}`, `{"puzzle":"` + strings.Repeat(".", 81) + `"} garbage`} {
			t.Run(path+body[:min(len(body), 20)], func(t *testing.T) {
				w := request(t, h, "POST", path, body)
				if w.Code != 400 {
					t.Fatalf("expected malformed request rejection, got %d: %s", w.Code, w.Body.String())
				}
				if w.Header().Get("Content-Type") != "application/json" || !json.Valid(w.Body.Bytes()) {
					t.Fatal("error must be JSON")
				}
			})
		}
	}
	w := request(t, h, "POST", "/api/validate", `{"puzzle":"`+strings.Repeat(".", 81)+`"}`)
	if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), `"issues":[]`) {
		t.Fatal("unexpected response headers or empty issues shape")
	}
}

func TestValidationCoordinates(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cells []int
	}{{"row", []int{0, 8}}, {"column", []int{0, 72}}, {"box", []int{0, 10}}} {
		t.Run(tt.name, func(t *testing.T) {
			puzzle := []byte(strings.Repeat(".", 81))
			for _, index := range tt.cells {
				puzzle[index] = '1'
			}
			issues := solver.ValidatePuzzle(string(puzzle))
			found := false
			for _, issue := range issues {
				if len(issue.Cells) == 2 && issue.Cells[0].Row*9+issue.Cells[0].Col == tt.cells[0] && issue.Cells[1].Row*9+issue.Cells[1].Col == tt.cells[1] {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing conflict cells: %+v", issues)
			}
		})
	}
	// R1C1 cannot take 1–8 (its row) or 9 (its column).
	puzzle := ".12345678" + "9........" + strings.Repeat(".", 63)
	w := request(t, NewHandler(nil), "POST", "/api/validate", `{"puzzle":"`+puzzle+`"}`)
	var response validationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Valid || !strings.Contains(w.Body.String(), "has no possible digits") {
		t.Fatalf("missed impossible cell: %s", w.Body.String())
	}
}

func TestHistoryMatchesEveryAppliedStep(t *testing.T) {
	puzzle := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	w := request(t, NewHandler(nil), "POST", "/api/solve", `{"puzzle":"`+puzzle+`"}`)
	var response solveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	s, err := solver.NewSudoku(solver.Puzzle{Puzzle: puzzle})
	if err != nil {
		t.Fatal(err)
	}
	for i, state := range response.States {
		if i > 0 {
			step, ok := s.SolveStep()
			if !ok || state.Step == nil || state.Step.Technique != step.Technique {
				t.Fatalf("step %d inconsistent", i)
			}
		}
		if state.Board != s.Board || state.Candidates != s.Candidates {
			t.Fatalf("snapshot %d inconsistent", i)
		}
	}
	expected := "534678912672195348198342567859761423426853791713924856961537284287419635345286179"
	for i, ch := range expected {
		if s.Board[i/9][i%9] != int8(ch-'0') {
			t.Fatal("incorrect final solution")
		}
	}
	w = request(t, NewHandler(nil), "POST", "/api/solve", `{"puzzle":"`+expected+`"}`)
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Solved || len(response.States) != 1 || response.States[0].Step != nil {
		t.Fatal("completed input should have only initial state")
	}
}

func TestRandomFailures(t *testing.T) {
	for _, source := range []func() (RandomPuzzle, error){
		func() (RandomPuzzle, error) { return RandomPuzzle{}, errors.New("private filesystem details") },
		func() (RandomPuzzle, error) { return RandomPuzzle{Puzzle: "11" + strings.Repeat(".", 79)}, nil },
	} {
		w := request(t, NewHandler(source), "GET", "/api/puzzles/random", "")
		if w.Code != 500 || !json.Valid(w.Body.Bytes()) || strings.Contains(w.Body.String(), "private filesystem") {
			t.Fatal("random errors must be handled without leaking internals")
		}
	}
}

func TestCatalogCSVAndConcurrentRequests(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.csv")
	puzzle := strings.Repeat(".", 81)
	// Quoted fields, CRLF, blank lines and missing final newline must preserve offsets.
	data := "id,puzzle,solution,clues,difficulty\r\n\r\n1," + puzzle + ",SECRET,0,\"easy, level 1\"\r\n2," + puzzle + ",SECRET,0,hard"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	h := NewHandler(catalog.Random)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				w := request(t, h, "GET", "/api/puzzles/random", "")
				var p RandomPuzzle
				if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
					t.Error(err)
					return
				}
				if w.Code != 200 || p.Puzzle != puzzle || (p.ID != "1" && p.ID != "2") {
					t.Errorf("concurrent read failed: %s", w.Body.String())
					return
				}
			}
		}()
	}
	wg.Wait()
	for _, data := range []string{"", "\"unclosed", "1,bad,secret,0,easy\n"} {
		path := filepath.Join(t.TempDir(), "bad.csv")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if c, err := OpenCatalog(path); err == nil {
			c.Close()
			t.Errorf("accepted unusable CSV %q", data)
		}
	}
	if c, err := OpenCatalog(filepath.Join(t.TempDir(), "missing.csv")); err == nil {
		c.Close()
		t.Fatal("missing file accepted")
	}
}

// Enable explicitly to inspect the local 3-million-puzzle database without
// making ordinary tests depend on a large external file.
func TestRealCatalog(t *testing.T) {
	path := os.Getenv("GODOKU_TEST_DATABASE")
	if path == "" {
		t.Skip("set GODOKU_TEST_DATABASE to test the real CSV")
	}
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	t.Logf("indexed %d puzzles", len(catalog.offsets))
	h := NewHandler(catalog.Random)
	solved, stuck := 0, 0
	for i := range 50 {
		w := request(t, h, "GET", "/api/puzzles/random", "")
		var p RandomPuzzle
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil || w.Code != 200 {
			t.Fatalf("random %d failed: %s", i, w.Body.String())
		}
		w = request(t, h, "POST", "/api/solve", fmt.Sprintf(`{"puzzle":%q}`, p.Puzzle))
		var result solveResponse
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 {
			t.Fatalf("solve %d failed: %s", i, w.Body.String())
		}
		for _, state := range result.States {
			var board strings.Builder
			for _, row := range state.Board {
				for _, value := range row {
					fmt.Fprintf(&board, "%d", value)
				}
			}
			if issues := solver.ValidatePuzzle(board.String()); len(issues) > 0 {
				t.Fatalf("puzzle %s produced invalid state: %+v", p.ID, issues)
			}
			for r := range 9 {
				for c := range 9 {
					if state.Board[r][c] == 0 && state.Candidates[r][c] == 0 {
						t.Fatal("empty cell has no candidates")
					}
				}
			}
		}
		last := result.States[len(result.States)-1]
		s := solver.Sudoku{Board: last.Board, Candidates: last.Candidates}
		if result.Solved != s.IsSolved() {
			t.Fatal("solved flag inconsistent")
		}
		if result.Solved {
			solved++
		} else {
			stuck++
		}
	}
	t.Logf("50 random API solves: %d solved, %d stuck", solved, stuck)
}
