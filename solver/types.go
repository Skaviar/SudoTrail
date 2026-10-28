package solver

type Puzzle struct {
	Id         string
	Puzzle     string
	Solution   string
	Clues      string
	Difficulty string
}

type Sudoku struct {
	initialBoard     [9][9]int8
	initialKnown     bool
	uniqueness       int8 // 0 = unchecked, 1 = unique, -1 = not unique
	pending          candidateUpdate
	deferPeerUpdates bool
	Board            [9][9]int8
	Candidates       [9][9]uint16
}
