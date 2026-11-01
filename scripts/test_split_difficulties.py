import csv
from pathlib import Path
import tempfile
import unittest

from split_difficulties import split_difficulties


class SplitDifficultiesTests(unittest.TestCase):
    def test_preserves_rows_columns_order_and_unrated_entries(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, destination = root / "rated.csv", root / "split"
            header = ["id", "difficulty", "rating_status", "technique_counts"]
            rows = [
                ["1,quoted", "easy", "solved", '{"Hidden Single":10,"Full House":2}'],
                ["2", "medium", "solved", '{"Naked Pair":1}'],
                ["3\nmultiline", "hard", "solved", '{"X-Wing":1}'],
                ["4", "expert", "solved", '{"3D Medusa":1}'],
                ["5", " EASY ", "solved", "{}"],
                ["6", "", "unsolved", "{}"],
                ["7", "hard", "invalid", "{}"],
            ]
            with source.open("w", encoding="utf-8-sig", newline="") as file:
                csv.writer(file).writerows([header, *rows])
            original = source.read_bytes()
            counts = split_difficulties(source, destination)
            self.assertEqual(counts, {"easy": 2, "medium": 1, "hard": 1, "expert": 1, "unrated": 2})
            expected = {"easy": [rows[0], rows[4]], "medium": [rows[1]], "hard": [rows[2]], "expert": [rows[3]], "unrated": rows[5:]}
            for level, selected in expected.items():
                with (destination / f"{level}.csv").open(newline="", encoding="utf-8") as file:
                    self.assertEqual(list(csv.reader(file)), [header, *selected])
            self.assertEqual(source.read_bytes(), original)

    def test_does_not_overwrite_any_existing_output_or_source(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "easy.csv"
            source.write_text("id,difficulty\n1,easy\n", encoding="utf-8")
            original = source.read_bytes()
            with self.assertRaises(FileExistsError):
                split_difficulties(source, root)
            self.assertEqual(source.read_bytes(), original)
            self.assertFalse((root / "medium.csv").exists())

    def test_validates_header_difficulties_and_row_shapes(self):
        for contents in ("", "id,original_difficulty\n1,1.5\n", "id,difficulty\n1,2.2\n", "id,difficulty\n1,easy,extra\n"):
            with self.subTest(contents=contents), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                source = root / "input.csv"
                source.write_text(contents, encoding="utf-8")
                with self.assertRaises(ValueError):
                    split_difficulties(source, root / "split")

    def test_creates_all_four_files_but_no_empty_unrated_file(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / "input.csv"
            source.write_text("id,difficulty\n1,hard\n", encoding="utf-8")
            counts = split_difficulties(source, root / "split")
            self.assertEqual(counts["hard"], 1)
            self.assertEqual(sorted(p.name for p in (root / "split").iterdir()), ["easy.csv", "expert.csv", "hard.csv", "medium.csv"])


if __name__ == "__main__":
    unittest.main()
