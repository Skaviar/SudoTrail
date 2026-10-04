package solver

import "math/bits"

const AllCandidates uint16 = 0b1111111110

func hasCandidate(mask uint16, number int) bool {
	return mask&(1<<number) != 0
}

func removeCandidate(mask uint16, number int) uint16 {
	return mask &^ (1 << number)
}

func candidateCount(mask uint16) int {
	return bits.OnesCount16(mask)
}

func singleCandidate(mask uint16) int {
	if candidateCount(mask) != 1 {
		return 0
	}

	for number := 1; number <= 9; number++ {
		if hasCandidate(mask, number) {
			return number
		}
	}

	return 0
}

func candidateList(mask uint16) []int {
	var result []int

	for number := 1; number <= 9; number++ {
		if hasCandidate(mask, number) {
			result = append(result, number)
		}
	}

	return result
}
