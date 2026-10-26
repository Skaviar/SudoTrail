package solver

type Puzzle struct {
	Id         string
	Puzzle     string
	Solution   string
	Clues      string
	Difficulty string
}

type Sudoku struct {
	pending          candidateUpdate
	deferPeerUpdates bool
	Board            [9][9]int8
	Candidates       [9][9]uint16
}
