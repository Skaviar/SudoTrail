package solver

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func candidateLiteralID(c CandidateLiteral) int {
	id := candidateID(c.Row*9+c.Col, c.Number) * 2
	if c.True {
		id++
	}
	return id
}

// Independently check each proof against the ORIGINAL exact-one constraints,
// rather than trusting a trace produced by the propagation implementation.
func assertProofBranches(t *testing.T, before BoardState, step Step) {
	t.Helper()
	s := &Sudoku{Board: before.Board, Candidates: before.Candidates}
	g := buildCandidateGraph(s)
	for _, branch := range step.Visual.Proofs {
		if len(branch.Nodes) == 0 {
			t.Fatal("empty proof branch")
		}
		for index, node := range branch.Nodes {
			if !cellHasCandidate(s, node.Candidate.cell(), node.Number) {
				t.Fatal("proof names an absent candidate")
			}
			if index == 0 {
				if len(node.Parents) != 0 {
					t.Fatal("assumption has parents")
				}
				continue
			}
			parents := map[int]bool{}
			for _, p := range node.Parents {
				if p < 0 || p >= index {
					t.Fatal("proof has forward or invalid dependency")
				}
				parents[candidateLiteralID(branch.Nodes[p].CandidateLiteral)] = true
			}
			id := candidateID(node.Row*9+node.Col, node.Number)
			valid := false
			if !node.True {
				for p := range parents {
					valid = valid || p%2 == 1 && candidatesConflict(p/2, id)
				}
			} else {
				for _, constraint := range g.constraints {
					contains, othersFalse := false, true
					for _, other := range constraint {
						if other == id {
							contains = true
						} else {
							othersFalse = othersFalse && parents[other*2]
						}
					}
					valid = valid || contains && othersFalse
				}
			}
			if !valid {
				t.Fatalf("unproved inference in %s: %+v", step.Technique, node)
			}
		}
		if branch.Contradiction != "" {
			conflict := map[int]bool{}
			for _, index := range branch.Conflict {
				if index < 0 || index >= len(branch.Nodes) {
					t.Fatal("invalid contradiction dependency")
				}
				conflict[candidateLiteralID(branch.Nodes[index].CandidateLiteral)] = true
			}
			valid := false
			for lit := range conflict {
				valid = valid || conflict[lit^1]
			}
			for _, constraint := range g.constraints {
				empty := true
				for _, id := range constraint {
					empty = empty && conflict[id*2]
				}
				valid = valid || empty
			}
			if !valid {
				t.Fatal("reported contradiction is not a contradiction")
			}
		}
	}
}

func assertInferenceLinks(t *testing.T, before BoardState, step Step) {
	t.Helper()
	for _, link := range step.Visual.Inferences {
		a, b := link.From, link.To
		if !hasCandidate(before.Candidates[a.Row][a.Col], a.Number) || !hasCandidate(before.Candidates[b.Row][b.Col], b.Number) {
			t.Fatal("inference links absent candidates")
		}
		if link.Kind == "implication" {
			if len(step.Visual.Proofs) == 0 {
				t.Fatal("net link has no proof DAG")
			}
			continue
		}
		if !candidatesConflict(candidateID(indexCell(a.cell()), a.Number), candidateID(indexCell(b.cell()), b.Number)) {
			t.Fatalf("invented weak/strong connection: %+v", link)
		}
		if link.Kind != "strong" {
			continue
		}
		strong := a.cell() == b.cell() && candidateCount(before.Candidates[a.Row][a.Col]) == 2
		if a.Number == b.Number {
			for _, house := range allUnits() {
				if !containsCell(house.cells, a.cell()) || !containsCell(house.cells, b.cell()) {
					continue
				}
				count := 0
				for _, c := range house.cells {
					if hasCandidate(before.Candidates[c.Row][c.Col], a.Number) {
						count++
					}
				}
				strong = strong || count == 2
			}
		}
		if !strong {
			t.Fatalf("invented strong link: %+v", link)
		}
	}
	assertProofBranches(t, before, step)
}

func TestAdvancedInferenceProofsAreValid(t *testing.T) {
	data, err := os.ReadFile("testdata/advanced_puzzles.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct{ Technique, Puzzle string }
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, f := range fixtures {
		s, _ := NewSudoku(Puzzle{Puzzle: f.Puzzle})
		history := s.Solve()
		for i, entry := range history.Entries {
			if i == 0 || entry.Step.Visual == nil {
				continue
			}
			assertInferenceLinks(t, history.Entries[i-1].State, entry.Step)
			counts[entry.Step.Technique] += len(entry.Step.Visual.Proofs)
		}
	}
	if counts["Forcing Net"] == 0 || counts["Forcing Chain"] == 0 {
		t.Fatal("forcing proofs were not exercised")
	}
}

func TestMultipleForcingNetBranchesHaveCompleteProofs(t *testing.T) {
	const puzzle = "4....87......2.....9..5.862..6..7.84..5.....7...86.12.3..1.......9.7...86..9..4.."
	s, _ := NewSudoku(Puzzle{Puzzle: puzzle})
	history := s.Solve()
	found := false
	for i, entry := range history.Entries {
		if entry.Step.Technique != "Forcing Net" || len(entry.Step.Visual.Proofs) < 2 {
			continue
		}
		found = true
		assertProofBranches(t, history.Entries[i-1].State, entry.Step)
		premises := map[int]bool{}
		for _, branch := range entry.Step.Visual.Proofs {
			premises[candidateLiteralID(branch.Nodes[0].CandidateLiteral)] = true
		}
		exhaustive := false
		for lit := range premises {
			exhaustive = exhaustive || premises[lit^1]
		}
		if !exhaustive {
			t.Fatal("branches do not cover both possibilities of the premise")
		}
	}
	if !found {
		t.Fatal("reference no longer exercises a multiple forcing proof")
	}
}

func TestAdvancedMethodsLeaveUnconstrainedPuzzleUntouched(t *testing.T) {
	for _, apply := range []func(*Sudoku) (Step, bool){ApplyXChains, ApplyXYChains, ApplyAIC, ApplyMedusa, ApplyMultiColoring, ApplyALS, ApplyDeathBlossom, ApplyAdvancedRectangles, ApplyMissingCandidateRectangles, ApplyAvoidableRectangle, ApplyForcingChains, ApplyForcingNets} {
		s, _ := NewSudoku(Puzzle{Puzzle: strings.Repeat(".", 81)})
		before := CaptureBoardState(s)
		if _, ok := apply(s); ok || CaptureBoardState(s) != before {
			t.Fatal("invented deduction on an empty puzzle")
		}
	}
}

func TestHistoryOwnsNestedProofDependencies(t *testing.T) {
	s := patternSudoku()
	history := NewSolverHistory(s)
	step := Step{Visual: &StepVisual{Proofs: []ProofBranch{{Nodes: []ProofNode{{Parents: []int{0}}}, Conflict: []int{0}}}, Inferences: []InferenceLink{{Kind: "weak"}}}}
	history.RecordStep(s, step)
	step.Visual.Proofs[0].Nodes[0].Parents[0] = 5
	step.Visual.Proofs[0].Conflict[0] = 5
	step.Visual.Inferences[0].Kind = "changed"
	if v := history.Entries[1].Step.Visual; v.Proofs[0].Nodes[0].Parents[0] != 0 || v.Proofs[0].Conflict[0] != 0 || v.Inferences[0].Kind != "weak" {
		t.Fatal("history shares mutable proof storage")
	}
}
