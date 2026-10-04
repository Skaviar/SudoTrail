package solver

import "math/bits"

const AllCandidates uint16 = 0b1111111110

func HasCandidate(mask uint16, number int) bool {
	return mask&(1<<number) != 0
}

func RemoveCandidate(mask uint16, number int) uint16 {
	return mask &^ (1 << number)
}

func CandidateCount(mask uint16) int {
	return bits.OnesCount16(mask)
}

func SingleCandidate(mask uint16) int {
	if CandidateCount(mask) != 1 {
		return 0
	}

	for number := 1; number <= 9; number++ {
		if HasCandidate(mask, number) {
			return number
		}
	}

	return 0
}

func CandidateList(mask uint16) []int {
	var result []int

	for number := 1; number <= 9; number++ {
		if HasCandidate(mask, number) {
			result = append(result, number)
		}
	}

	return result
}
