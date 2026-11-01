#!/usr/bin/env python3
"""Stream a rated Sudoku CSV into separate files using only Python's stdlib."""

import argparse
import csv
from contextlib import ExitStack
from pathlib import Path


LEVELS = ("easy", "medium", "hard", "expert")


def split_difficulties(source: Path, output_dir: Path) -> dict[str, int]:
    counts = dict.fromkeys((*LEVELS, "unrated"), 0)
    with source.open("r", encoding="utf-8-sig", newline="") as input_file:
        reader = csv.reader(input_file, strict=True)
        header = next(reader, None)
        if not header:
            raise ValueError("input CSV is empty")
        columns = [name.strip().lower() for name in header]
        if columns.count("difficulty") != 1:
            raise ValueError("input must have exactly one 'difficulty' column")
        if columns.count("rating_status") > 1:
            raise ValueError("input has duplicate 'rating_status' columns")
        difficulty_column = columns.index("difficulty")
        status_column = columns.index("rating_status") if "rating_status" in columns else None

        # Check every destination before creating any files, then use exclusive
        # creation as well so existing exports and source aliases stay intact.
        paths = {level: output_dir / f"{level}.csv" for level in counts}
        for path in paths.values():
            if path.exists() or path.is_symlink():
                raise FileExistsError(f"{path} already exists; choose a new --output-dir")
        output_dir.mkdir(parents=True, exist_ok=True)
        with ExitStack() as files:
            writers = {}

            def open_writer(level: str):
                handle = files.enter_context(paths[level].open("x", encoding="utf-8", newline=""))
                writer = csv.writer(handle, lineterminator="\n")
                writer.writerow(header)
                writers[level] = writer
                return writer

            for level in LEVELS:
                open_writer(level)

            processed = 0
            for row in reader:
                if not row:
                    continue
                if len(row) != len(header):
                    raise ValueError(f"CSV line {reader.line_num}: expected {len(header)} columns, got {len(row)}")
                level = row[difficulty_column].strip().lower()
                status = row[status_column].strip().lower() if status_column is not None else ""
                if status not in ("", "solved", "unsolved", "invalid"):
                    raise ValueError(f"CSV line {reader.line_num}: unknown rating status {status!r}")
                if status in ("unsolved", "invalid") or not level:
                    level = "unrated"
                elif level not in LEVELS:
                    raise ValueError(
                        f"CSV line {reader.line_num}: unknown difficulty {level!r}; use the rated CSV"
                    )
                writer = writers.get(level)
                if writer is None:
                    writer = open_writer(level)
                writer.writerow(row)
                counts[level] += 1
                processed += 1
                if processed % 250_000 == 0:
                    print(f"Processed {processed:,} puzzles...", flush=True)
    return counts


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", nargs="?", type=Path, default=Path("sudoku-rated.csv"), help="rated CSV (default: sudoku-rated.csv)")
    parser.add_argument("--output-dir", type=Path, default=Path("puzzles-by-difficulty"), help="destination directory (default: puzzles-by-difficulty)")
    args = parser.parse_args()
    try:
        counts = split_difficulties(args.input, args.output_dir)
    except (OSError, ValueError, csv.Error, UnicodeError) as error:
        parser.exit(1, f"Error: {error}\n")
    except KeyboardInterrupt:
        parser.exit(130, f"Stopped. Partial files remain in {args.output_dir}.\n")
    print(f"\nSplit {sum(counts.values()):,} puzzles:")
    for level, count in counts.items():
        if level != "unrated" or count:
            print(f"{level.capitalize():<8} {count:>10,} -> {args.output_dir / (level + '.csv')}")


if __name__ == "__main__":
    main()
