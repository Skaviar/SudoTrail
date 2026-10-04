package solver

type Puzzle struct {
	Id         string
	Puzzle     string
	Solution   string
	Clues      string
	Difficulty string
}

type Sudoku struct {
	Board      [9][9]int8
	Candidates [9][9]uint16
}

type Step struct {
	Technique        string
	Explanation      string
	Details          []string
	Row              int
	Col              int
	Number           int
	BeforeCandidates uint16
}
