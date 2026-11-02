package webapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/s-threepio/godoku/internal/puzzleapp"
	"github.com/s-threepio/godoku/internal/puzzlebank"
	"github.com/s-threepio/godoku/solver"
)

type puzzleRequest struct {
	Puzzle string `json:"puzzle"`
}
type validationResponse = puzzleapp.ValidationResponse
type stateResponse = puzzleapp.StateResponse
type solveResponse = puzzleapp.SolveResponse

// NewHandler exposes only puzzle APIs; benchmark execution belongs to the CLI.
// A nil random source leaves manual validation and solving available.
func NewHandler(random func() (RandomPuzzle, error)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/puzzles/catalog", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, puzzlebank.Catalog())
	})
	mux.HandleFunc("POST /api/validate", func(w http.ResponseWriter, r *http.Request) {
		request, ok := readPuzzle(w, r)
		if !ok {
			return
		}
		issues := solver.ValidatePuzzle(request.Puzzle)
		writeJSON(w, http.StatusOK, validationResponse{Valid: len(issues) == 0, Issues: issues})
	})
	mux.HandleFunc("POST /api/solve", func(w http.ResponseWriter, r *http.Request) {
		request, ok := readPuzzle(w, r)
		if !ok {
			return
		}
		if issues := solver.ValidatePuzzle(request.Puzzle); len(issues) > 0 {
			writeJSON(w, http.StatusBadRequest, validationResponse{Valid: false, Issues: issues})
			return
		}
		response, err := puzzleapp.Solve(request.Puzzle)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("GET /api/puzzles/random", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("source") == "practice" || query.Get("difficulty") != "" || query.Get("technique") != "" {
			puzzle, err := puzzlebank.Select(puzzlebank.Selection{Difficulty: query.Get("difficulty"), Technique: query.Get("technique"), Exclude: query.Get("exclude")})
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, puzzle)
			return
		}
		if random == nil {
			writeError(w, http.StatusServiceUnavailable, "random puzzle database is unavailable")
			return
		}
		puzzle, err := random()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load random puzzle")
			return
		}
		if len(solver.ValidatePuzzle(puzzle.Puzzle)) > 0 {
			writeError(w, http.StatusInternalServerError, "database puzzle has invalid clues")
			return
		}
		writeJSON(w, http.StatusOK, puzzle)
	})
	return mux
}

func readPuzzle(w http.ResponseWriter, r *http.Request) (puzzleRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request puzzleRequest
	var payload *struct {
		Puzzle *string `json:"puzzle"`
	}
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "expected JSON with a puzzle string (maximum 4096 bytes)")
		return request, false
	}
	if payload == nil || payload.Puzzle == nil {
		writeError(w, http.StatusBadRequest, "expected a JSON object with a puzzle string")
		return request, false
	}
	request.Puzzle = *payload.Puzzle
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "expected one JSON object")
		return request, false
	}
	return request, true
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
