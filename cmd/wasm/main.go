//go:build js && wasm

// The browser worker calls this entry point; no HTTP server is used.
package main

import (
	"encoding/json"
	"net/url"
	"syscall/js"

	"github.com/s-threepio/godoku/internal/puzzleapp"
	"github.com/s-threepio/godoku/internal/puzzlebank"
)

type response struct {
	Status int `json:"status"`
	Body   any `json:"body"`
}

func request(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if recover() != nil {
			result = `{"status":500,"body":{"error":"The browser solver failed. Reload the page and try again."}}`
		}
	}()
	if len(args) != 2 {
		return `{"status":400,"body":{"error":"invalid solver request"}}`
	}
	path, puzzle := args[0].String(), args[1].String()
	parsed, parseErr := url.Parse(path)
	if parseErr != nil {
		return `{"status":400,"body":{"error":"invalid solver operation"}}`
	}
	reply := response{Status: 200}
	switch parsed.Path {
	case "/api/validate":
		reply.Body = puzzleapp.Validate(puzzle)
	case "/api/solve", "/api/hint":
		validation := puzzleapp.Validate(puzzle)
		if !validation.Valid {
			reply.Status = 400
			reply.Body = validation
			break
		}
		var solved any
		var err error
		if parsed.Path == "/api/hint" {
			solved, err = puzzleapp.Hint(puzzle)
		} else {
			solved, err = puzzleapp.Solve(puzzle)
		}
		if err != nil {
			reply.Status = 400
			reply.Body = map[string]string{"error": err.Error()}
		} else {
			reply.Body = solved
		}
	case "/api/puzzles/random":
		query := parsed.Query()
		selected, err := puzzlebank.Select(puzzlebank.Selection{Difficulty: query.Get("difficulty"), Technique: query.Get("technique"), Exclude: query.Get("exclude")})
		if err != nil {
			reply.Status = 400
			reply.Body = map[string]string{"error": err.Error()}
		} else {
			reply.Body = selected
		}
	case "/api/puzzles/catalog":
		reply.Body = puzzlebank.Catalog()
	default:
		reply.Status = 404
		reply.Body = map[string]string{"error": "unknown solver operation"}
	}
	encoded, err := json.Marshal(reply)
	if err != nil {
		return `{"status":500,"body":{"error":"Could not encode solver result"}}`
	}
	return string(encoded)
}
func main() {
	callback := js.FuncOf(request)
	js.Global().Set("godokuRequest", callback)
	js.Global().Call("godokuReady")
	select {}
}
