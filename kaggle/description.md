This dataset contains **3,000,000 Sudoku puzzles and their solutions**, enriched with technique-based difficulty ratings from the open-source **Sudotrail** solver.

The original puzzles come from [David Radcliffe's 3 million Sudoku puzzles with ratings](https://www.kaggle.com/datasets/radcliffe/3-million-sudoku-puzzles-with-ratings). This release adds new difficulty labels, weighted scores, hardest-technique labels, logical step counts, and technique usage counts. The original difficulty values are preserved in `original_difficulty`.

## Difficulty distribution

| Difficulty | Puzzles | Typical techniques |
| --- | ---: | --- |
| Easy | 1,294,081 | Full House, Naked Single, Hidden Single |
| Medium | 557,035 | Locked Candidates, naked/hidden pairs and triples |
| Hard | 511,762 | Quads, basic fish, wings, coloring, and basic uniqueness patterns |
| Expert | 637,122 | Finned/complex fish, advanced rectangles, chains, ALS, and forcing techniques |
| **Total** | **3,000,000** | |

All rows in this release have `rating_status=solved` and a difficulty label.

## How the ratings work

The solver starts from the puzzle's clues and tries easier technique tiers before harder ones. After every deduction, it tries the easier tiers again.

- `difficulty` records the hardest technique tier used in the solving path.
- `score` sums the weights of the logical deductions, measuring the amount of work within a tier.
- `logical_steps` counts placements and elimination patterns. Automatic candidate cleanup after placing a number contributes no steps or points.
- `technique_counts` records how often each technique was used.

The rating policy is versioned as **`godoku-v1`**. Ratings are specific to Sudotrail and the solving path it finds; they do not prove that no easier path exists. The supplied solution and original difficulty are not used to generate the new ratings. Backtracking is used only to check uniqueness for techniques that require it.

## File and columns

The dataset contains `sudoku-rated.csv`, with one puzzle per row.

| Column | Meaning |
| --- | --- |
| `id` | Identifier retained from the original dataset |
| `puzzle` | 81 characters in row order; digits 1–9 are clues and `.` represents a blank |
| `solution` | Supplied completed grid as an 81-character string |
| `clues` | Number of given digits |
| `difficulty` | New label: `easy`, `medium`, `hard`, or `expert` |
| `original_difficulty` | Difficulty value retained from the source |
| `score` | Sum of logical technique weights |
| `hardest_technique` | Highest-rated technique used in the solving path |
| `logical_steps` | Number of logical deductions |
| `technique_counts` | JSON object mapping technique names to usage counts |
| `rating_status` | Solver outcome; `solved` for all rows in this release |
| `rating_version` | Rating policy identifier: `godoku-v1` |
| `rating_error` | Error details, when present; empty for this release |

## Loading the data

Treat puzzle and solution values as strings. For example, with pandas:

```python
import json
import pandas as pd

df = pd.read_csv("sudoku-rated.csv", dtype={"puzzle": "string", "solution": "string"})
counts = json.loads(df.loc[0, "technique_counts"])
```

The dataset can support Sudoku solver benchmarking, difficulty classification, technique analysis, and puzzle selection for practice applications.

## Credits and license

- **Original puzzles and solutions:** [David Radcliffe](https://www.kaggle.com/datasets/radcliffe/3-million-sudoku-puzzles-with-ratings).
- **New ratings and annotations:** Swanand Kavitkar, using [Sudotrail](https://github.com/Skaviar/Godoku).
- **Rating implementation:** [solver/rating.go](https://github.com/Skaviar/Godoku/blob/main/solver/rating.go).
- **License:** [CC0: Public Domain](https://creativecommons.org/publicdomain/zero/1.0/), matching the source dataset.
