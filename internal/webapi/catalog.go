package webapi

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
)

type RandomPuzzle struct {
	ID         string `json:"id"`
	Puzzle     string `json:"puzzle"`
	Difficulty string `json:"difficulty"`
}

// Catalog keeps byte offsets instead of retaining millions of CSV rows.
// ReadAt allows concurrent requests to use the same file safely.
type Catalog struct {
	file    *os.File
	offsets []int64
}

func OpenCatalog(path string) (*Catalog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{file: file}
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	for {
		offset := reader.InputOffset()
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("read puzzle database: %w", err)
		}
		if len(row) < 5 || row[0] == "id" || len(row[1]) != 81 {
			continue
		}
		valid := true
		for _, ch := range row[1] {
			if ch != '.' && (ch < '0' || ch > '9') {
				valid = false
				break
			}
		}
		if valid {
			catalog.offsets = append(catalog.offsets, offset)
		}
	}
	if len(catalog.offsets) == 0 {
		file.Close()
		return nil, fmt.Errorf("database contains no usable puzzles")
	}
	return catalog, nil
}

func (c *Catalog) Close() error { return c.file.Close() }

func (c *Catalog) Random() (RandomPuzzle, error) {
	offset := c.offsets[rand.IntN(len(c.offsets))]
	reader := csv.NewReader(io.NewSectionReader(c.file, offset, 1<<20))
	row, err := reader.Read()
	if err != nil {
		return RandomPuzzle{}, err
	}
	if len(row) < 5 {
		return RandomPuzzle{}, fmt.Errorf("invalid database record")
	}
	return RandomPuzzle{ID: row[0], Puzzle: row[1], Difficulty: row[4]}, nil
}
