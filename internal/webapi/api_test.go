package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s-threepio/godoku/internal/puzzlebank"
)

func request(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestPracticeCatalogAndFilteredPuzzlesWithoutCSV(t *testing.T) {
	h := NewHandler(nil)
	w := request(t, h, "GET", "/api/puzzles/catalog", "")
	var levels []puzzlebank.Level
	if err := json.Unmarshal(w.Body.Bytes(), &levels); err != nil || w.Code != 200 || len(levels) != 4 {
		t.Fatal("missing practice catalog")
	}
	for _, level := range levels {
		query := url.Values{"source": {"practice"}, "difficulty": {level.Difficulty}, "technique": {level.Techniques[0].Name}}
		w := request(t, h, "GET", "/api/puzzles/random?"+query.Encode(), "")
		var p puzzlebank.Puzzle
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil || w.Code != 200 || p.Difficulty != level.Difficulty {
			t.Fatalf("wrong practice puzzle: %s", w.Body.String())
		}
		solved := request(t, h, "POST", "/api/solve", `{"puzzle":"`+p.Puzzle+`"}`)
		var history solveResponse
		if err := json.Unmarshal(solved.Body.Bytes(), &history); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, state := range history.States {
			found = found || state.Step != nil && state.Step.Technique == query.Get("technique")
		}
		if !found {
			t.Fatal("requested practice technique is missing from the actual web history")
		}
	}
	for _, query := range []string{"difficulty=impossible", "difficulty=easy&technique=Forcing+Net"} {
		if w := request(t, h, "GET", "/api/puzzles/random?"+query, ""); w.Code != 400 {
			t.Fatal("invalid practice filter was ignored")
		}
	}
}
func TestValidation(t *testing.T) {
	h := NewHandler(nil)
	for _, tt := range []struct {
		puzzle string
		valid  bool
	}{{strings.Repeat(".", 81), true}, {"11" + strings.Repeat(".", 79), false}, {"123", false}, {"x" + strings.Repeat(".", 80), false}} {
		w := request(t, h, "POST", "/api/validate", `{"puzzle":"`+tt.puzzle+`"}`)
		var response validationResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || response.Valid != tt.valid {
			t.Fatalf("unexpected validation: %s", w.Body.String())
		}
		if !tt.valid && len(response.Issues) == 0 {
			t.Fatal("missing validation errors")
		}
		if strings.HasPrefix(tt.puzzle, "11") && len(response.Issues[0].Cells) != 2 {
			t.Fatal("must identify both duplicate clues")
		}
	}
}
func TestSolve(t *testing.T) {
	h := NewHandler(nil)
	puzzle := "530070000600195000098000060800060003400803001700020006060000280000419005000080079"
	w := request(t, h, "POST", "/api/solve", `{"puzzle":"`+puzzle+`"}`)
	var response solveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !response.Solved || response.Status != "solved" || len(response.States) < 2 {
		t.Fatalf("unexpected solve: %s", w.Body.String())
	}
	if response.States[0].Step != nil || response.States[0].Board[0][2] != 0 {
		t.Fatal("initial board must be preserved")
	}
	if response.States[1].Step == nil || !strings.Contains(w.Body.String(), `"technique"`) {
		t.Fatal("missing step details")
	}
	w = request(t, h, "POST", "/api/solve", `{"puzzle":"`+strings.Repeat(".", 81)+`"}`)
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Solved || response.Status != "stuck" || len(response.States) != 1 {
		t.Fatal("stuck should differ from solved")
	}
	if w := request(t, h, "POST", "/api/solve", `{"puzzle":"invalid"}`); w.Code != 400 {
		t.Fatal("invalid solve accepted")
	}
}
func TestRequestErrors(t *testing.T) {
	h := NewHandler(nil)
	for _, body := range []string{`{`, `{"puzzle":42}`, `{"puzzle":""} {}`, `{"unknown":true}`, `{"puzzle":"` + strings.Repeat(".", 5000) + `"}`} {
		if w := request(t, h, "POST", "/api/validate", body); w.Code != 400 {
			t.Errorf("accepted malformed request %q", body)
		}
	}
	if w := request(t, h, "GET", "/api/solve", ""); w.Code != 405 {
		t.Fatal("wrong method accepted")
	}
	if w := request(t, h, "GET", "/api/benchmark", ""); w.Code != 404 {
		t.Fatal("benchmark exposed")
	}
	if w := request(t, h, "GET", "/api/puzzles/random", ""); w.Code != 503 {
		t.Fatal("missing database should return 503")
	}
}
func TestCatalogAndRandom(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.csv")
	puzzle := strings.Repeat(".", 81)
	data := "id,puzzle,solution,clues,difficulty\n1," + puzzle + ",SECRET,0,1.0\n2," + puzzle + ",SECRET,0,2.0\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	catalog, err := OpenCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	if len(catalog.offsets) != 2 {
		t.Fatal("header should not be indexed")
	}
	h := NewHandler(catalog.Random)
	for range 10 {
		w := request(t, h, "GET", "/api/puzzles/random", "")
		var response RandomPuzzle
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || response.Puzzle != puzzle || response.ID == "" || strings.Contains(w.Body.String(), "SECRET") {
			t.Fatalf("bad random response: %s", w.Body.String())
		}
	}
	empty := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(empty, []byte("id,puzzle,solution,clues,difficulty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if c, err := OpenCatalog(empty); err == nil {
		c.Close()
		t.Fatal("empty catalog accepted")
	}
}
