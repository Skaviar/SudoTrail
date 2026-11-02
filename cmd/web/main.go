package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/s-threepio/godoku/internal/webapi"
	"github.com/s-threepio/godoku/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	database := flag.String("database", "sudoku-3m.csv", "puzzle CSV path")
	static := flag.String("static", "", "serve a built static WASM directory instead of the Go API")
	flag.Parse()
	if *static != "" {
		log.Printf("Sudotrail static site listening on http://%s", *addr)
		server := &http.Server{Addr: *addr, Handler: http.FileServer(http.Dir(*static)), ReadHeaderTimeout: 5 * time.Second}
		log.Fatal(server.ListenAndServe())
		return
	}
	var random func() (webapi.RandomPuzzle, error)
	catalog, err := webapi.OpenCatalog(*database)
	if err != nil {
		log.Printf("Random puzzles unavailable: %v. Manual puzzle APIs remain available.", err)
	} else {
		defer catalog.Close()
		random = catalog.Random
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", webapi.NewHandler(random))
	mux.Handle("/", web.Handler())
	server := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Sudotrail API listening on http://%s", *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
