#!/usr/bin/env python3
"""Regression tests for the format-aware contract validator."""

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


HERE = Path(__file__).resolve().parent
VALIDATOR = HERE / "validate.py"
FIXTURES = HERE / "fixtures"

CONTRACT_FILES = [
    "canonical-scan.sample.json",
    "usage-graph.sample.json",
    "localisation.sample.json",
    "decision-results.sample.json",
    "vex-decision.sample.json",
]


class VexFormatValidatorTests(unittest.TestCase):
    def make_fixture_dir(self):
        tmp = tempfile.TemporaryDirectory()
        base = Path(tmp.name)

        for name in CONTRACT_FILES:
            shutil.copy(FIXTURES / name, base / name)

        return tmp, base

    def run_validator(self, base):
        return subprocess.run(
            [sys.executable, str(VALIDATOR), "--dir", str(base)],
            text=True,
            capture_output=True,
            check=False,
        )

    def load_json(self, path):
        with open(path) as f:
            return json.load(f)

    def save_json(self, path, data):
        with open(path, "w") as f:
            json.dump(data, f, indent=2)
            f.write("\n")

    def test_each_supported_format_accepts_its_own_vocabulary(self):
        for fmt, token in (
            ("openvex", "under_investigation"),
            ("cyclonedx", "in_triage"),
        ):
            with self.subTest(format=fmt):
                tmp, base = self.make_fixture_dir()
                try:
                    decisions_path = base / "decision-results.sample.json"
                    decisions = self.load_json(decisions_path)

                    for decision in decisions["decisions"]:
                        mapping = decision.get("vexMapping", {})
                        if mapping.get("statement") == "under_investigation":
                            mapping["statement"] = token

                    self.save_json(decisions_path, decisions)

                    vex_path = base / "vex-decision.sample.json"
                    vex = self.load_json(vex_path)
                    vex["format"] = fmt
                    vex["investigationState"] = token
                    self.save_json(vex_path, vex)

                    result = self.run_validator(base)

                    self.assertEqual(
                        result.returncode,
                        0,
                        msg=result.stdout + result.stderr,
                    )
                finally:
                    tmp.cleanup()

    def test_mixed_vex_vocabularies_are_rejected(self):
        tmp, base = self.make_fixture_dir()
        try:
            path = base / "decision-results.sample.json"
            decisions = self.load_json(path)

            changed = False
            for decision in decisions["decisions"]:
                mapping = decision.get("vexMapping", {})
                if mapping.get("statement") == "under_investigation":
                    mapping["statement"] = "in_triage"
                    changed = True
                    break

            self.assertTrue(changed)
            self.save_json(path, decisions)

            result = self.run_validator(base)

            self.assertNotEqual(result.returncode, 0)
            self.assertIn(
                "VEX vocabularies are not mixed",
                result.stdout + result.stderr,
            )
        finally:
            tmp.cleanup()

    def test_not_affected_without_manual_reviewer_is_rejected(self):
        tmp, base = self.make_fixture_dir()
        try:
            path = base / "decision-results.sample.json"
            decisions = self.load_json(path)

            mapping = decisions["decisions"][0]["vexMapping"]
            mapping["statement"] = "not_affected"
            mapping.pop("manuallyReviewedBy", None)

            self.save_json(path, decisions)

            result = self.run_validator(base)

            self.assertNotEqual(result.returncode, 0)
            self.assertIn(
                "no automated not_affected",
                result.stdout + result.stderr,
            )
        finally:
            tmp.cleanup()


if __name__ == "__main__":
    unittest.main()
