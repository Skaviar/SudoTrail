package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/s-threepio/godoku/solver"
)

func main() {

	file, err := os.Open("sudoku-3m.csv")

	if err != nil {
		slog.Error("readfile err", slog.Any("error", err))
	}

	defer file.Close()

	fReader := bufio.NewScanner(file)
	var Puzzles []solver.Puzzle
	for fReader.Scan() {
		line := fReader.Text()
		appendPuzzle(&Puzzles, line)
	}
	sudoku := &solver.Sudoku{}
	fmt.Println(Puzzles[5].Puzzle)
	sudoku.InitializeSudoku(Puzzles[5])
	sudoku.Print()
	sudoku.Solve()
	fmt.Print("\n\n**************After solving**************\n\n")
	sudoku.Print()

}

func appendPuzzle(puzzles *[]solver.Puzzle, dataLine string) {
	dataArray := strings.Split(dataLine, ",")
	puzzleItem := solver.Puzzle{
		Id:         dataArray[0],
		Puzzle:     dataArray[1],
		Solution:   dataArray[2],
		Clues:      dataArray[3],
		Difficulty: dataArray[4],
	}
	*puzzles = append(*puzzles, puzzleItem)
}
