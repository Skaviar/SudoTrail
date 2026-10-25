//go:build js && wasm

// The browser worker calls this entry point; no HTTP server is used.
package main

import (
	"encoding/json"
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
	reply := response{Status: 200}
	switch path {
	case "/api/validate":
		reply.Body = puzzleapp.Validate(puzzle)
	case "/api/solve":
		validation := puzzleapp.Validate(puzzle)
		if !validation.Valid {
			reply.Status = 400
			reply.Body = validation
			break
		}
		solved, err := puzzleapp.Solve(puzzle)
		if err != nil {
			reply.Status = 400
			reply.Body = map[string]string{"error": err.Error()}
		} else {
			reply.Body = solved
		}
	case "/api/puzzles/random":
		reply.Body = puzzlebank.Random()
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
