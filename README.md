# Sudotrail

A Sudoku solver that explains each logical move, built with Go, WebAssembly,
and vanilla JavaScript.

**[Try it online](https://sudotrail.pages.dev/)**

- Enter a puzzle or import a photo (beta).
- Follow each placement and candidate elimination with visual explanations.
- Practice specific techniques across 2,000 Easy, Medium, Hard, and Expert puzzles.
- Pick up where you left off with progress saved in your browser.

The solver uses logical techniques. Backtracking is used only to check uniqueness.

## Run locally

Requires **Go 1.25.6+** and `make`. From the project root:

```sh
make dev
```

Open **http://127.0.0.1:8080**. This builds the Go solver to WASM and serves the
website; solving runs in your browser. No puzzle CSV is needed.
Run the command again after changing source files.

## Commands

Run `make` for the full command list.

| Command | Purpose |
| --- | --- |
| `make build` | Build the static website into `dist/` |
| `make web` | Run the website with the local Go HTTP API |
| `make benchmark WORKERS=8` | Benchmark a puzzle CSV with 8 workers |
| `make rate WORKERS=8` | Export difficulty ratings to `sudoku-rated.csv` |
| `make bank PER_LEVEL=500` | Rebuild the browser's practice library |
| `make check` | Run Go tests and `go vet` |
| `make race` | Run Go tests with race detection |
| `make fmt` | Format Go source |

Benchmarking and rating read `sudoku-3m.csv` by default. Set `DATABASE=path.csv`
to use another file, or `LIMIT=10000` for a short run. Rating exports require a
new output path; set `OUTPUT=another-rated.csv` if the default file exists.

## Deploy

For **Cloudflare Pages** Git integration, use build command `make build` and
output directory `dist`. The deployed site runs the solver in the browser;
hosting only needs to serve static files.

## Documentation

- [Data tools](docs/data-tools.md): benchmarks, difficulty ratings, CSV splitting, and practice library generation.
- [Build and deployment](docs/deployment.md): WASM, Cloudflare Pages, and build modes.
- [Web interface and API](docs/web.md): photo import, saved progress, and API requests.
- [Solver reference](docs/solver.md): code structure, logical techniques, and using the Go package.

## License

[MIT](LICENSE). Dataset provenance and licensing are documented in the
[Kaggle description](kaggle/description.md).
