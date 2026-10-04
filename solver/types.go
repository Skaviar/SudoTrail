package solver

import "fmt"

type Puzzle struct {
	Id         string
	Puzzle     string
	Solution   string
	Clues      string
	Difficulty string
}

type Sudoku struct {
	Rows             [9][9]int32
	Columns          [9][9]int32
	Boxes            [9][9]int32
	RowsRemaining    [9][10]bool
	ColumnsRemaining [9][10]bool
	BoxesRemaining   [9][10]bool
}

func (s *Sudoku) InitializeSudoku(p Puzzle) {
	for i := range 9 {
		for j := range 9 {
			index := i*9 + j

			var value int32

			if p.Puzzle[index] == '.' {
				value = 0
			} else {
				value = int32(p.Puzzle[index] - '0')
			}

			s.Rows[i][j] = value
			s.Columns[j][i] = value

			box := (i/3)*3 + (j / 3)
			boxIndex := (i%3)*3 + (j % 3)
			s.Boxes[box][boxIndex] = value
		}
	}

	// Initially, every number 1-9 is available.
	for i := range 9 {
		for num := 1; num <= 9; num++ {
			s.RowsRemaining[i][num] = true
			s.ColumnsRemaining[i][num] = true
			s.BoxesRemaining[i][num] = true
		}
	}

	// Remove numbers already present in the puzzle.
	for i := range 9 {
		for j := range 9 {
			value := s.Rows[i][j]

			if value == 0 {
				continue
			}

			s.RowsRemaining[i][value] = false
			s.ColumnsRemaining[j][value] = false

			box := (i/3)*3 + (j / 3)
			s.BoxesRemaining[box][value] = false
		}
	}
}
func (s *Sudoku) Print() {
	fmt.Println("+-------+-------+-------+")

	for i := range 9 {
		fmt.Print("| ")

		for j := range 9 {
			value := s.Rows[i][j]

			if value == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", value)
			}

			if j == 2 || j == 5 {
				fmt.Print("| ")
			}
		}

		fmt.Println("|")

		if i == 2 || i == 5 {
			fmt.Println("+-------+-------+-------+")
		}
	}

	fmt.Println("+-------+-------+-------+")
}
