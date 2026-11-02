package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
)

func main() {
	workers := flag.Int("workers", runtime.GOMAXPROCS(0), "number of benchmark or difficulty-rating workers")
	benchmark := flag.Bool("benchmark", false, "benchmark the puzzle database")
	rate := flag.Bool("rate", false, "rate the database and export difficulties")
	destination := flag.String("output", "sudoku-rated.csv", "new CSV path for difficulty export (must not exist)")
	database := flag.String("database", "sudoku-3m.csv", "puzzle CSV path")
	limit := flag.Int("limit", 0, "benchmark or rate the first N puzzle records (0 = all)")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Godoku commands:")
		fmt.Fprintln(flag.CommandLine.Output(), "  make dev                            Run the web UI locally")
		fmt.Fprintln(flag.CommandLine.Output(), "  go run . -benchmark [-workers N]    Benchmark the database")
		fmt.Fprintln(flag.CommandLine.Output(), "  go run . -rate [-output path.csv]   Export difficulty ratings")
		fmt.Fprintln(flag.CommandLine.Output(), "\nOptions:")
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(os.Args) == 1 {
		flag.CommandLine.SetOutput(os.Stdout)
		flag.Usage()
		return
	}
	if err := validateBenchmarkOptions(*workers, *limit); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments; use -help for available flags")
		os.Exit(2)
	}
	if *rate && *benchmark {
		fmt.Fprintln(os.Stderr, "choose either -rate or -benchmark")
		os.Exit(2)
	}
	if *rate {
		if err := runRatingExport(*database, *destination, *workers, *limit, os.Stdout); err != nil {
			slog.Error("difficulty export failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}
	if *benchmark {
		if err := runBenchmark(*database, *workers, *limit, os.Stdout); err != nil {
			slog.Error("benchmark failed", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	fmt.Fprintln(os.Stderr, "choose -benchmark or -rate; use make dev for the web UI")
	os.Exit(2)
}
