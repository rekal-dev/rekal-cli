"""Regressions for the current public CLI contract used by benchmark smoke runs."""

import importlib.util
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch


PATH = Path(__file__).resolve().parents[1] / "industry-bench" / "shim" / "shim.py"
SPEC = importlib.util.spec_from_file_location("industry_shim", PATH)
SHIM = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(SHIM)


class IndustryShimTests(unittest.TestCase):
    def test_json_is_explicit_and_shipped_digest_controls_gate(self):
        results = [
            {"session_id": "s1", "confidence": 0.4, "snippet": "first hit"},
            {"session_id": "s2", "confidence": 0.8, "snippet": "second hit"},
        ]
        payload = {"results": results, "knowledge": [{"path": "docs/current.md", "snippet": "current prose"}]}
        calls = []

        def run(args, **kwargs):
            calls.append(args)
            output = json.dumps(payload) if "--json" in args else "INJECT top=0.80 gap=0.40 2 seeds\nKNOWLEDGE docs/current.md=0.8\n"
            return subprocess.CompletedProcess(args, 0, output, "")

        with patch.object(SHIM.subprocess, "run", side_effect=run):
            value = SHIM.search(Path("/bin/rekal"), Path("/repo"), "log", env={}, route="skill")
        self.assertEqual(value["route"]["gate"], "INJECT")
        self.assertEqual([c["session_id"] for c in value["contexts"]], ["s1", "s2"])
        self.assertEqual(len(value["knowledge"]), 1)
        self.assertEqual(value["recall_calls"], 2)
        self.assertIn("--json", calls[0])
        self.assertEqual(calls[0][-2:], ["--", "log"])
        self.assertEqual(calls[1][0], "/bin/rekal")
        self.assertNotIn("--json", calls[1])

    def test_silence_exit_one_is_a_valid_result(self):
        replies = [
            subprocess.CompletedProcess([], 0, '{"results":[],"knowledge":[]}', ""),
            subprocess.CompletedProcess([], 1, "SILENCE reason=no_results\n", ""),
        ]
        with patch.object(SHIM.subprocess, "run", side_effect=replies):
            value = SHIM.search(Path("/bin/rekal"), Path("/repo"), "missing", env={}, route="skill")
        self.assertEqual(value["route"]["gate"], "SILENCE")
        self.assertEqual(value["contexts"], [])

    def test_command_failure_is_not_reported_as_silence(self):
        reply = subprocess.CompletedProcess([], 1, "", "database unavailable")
        with patch.object(SHIM.subprocess, "run", return_value=reply):
            with self.assertRaisesRegex(RuntimeError, "database unavailable"):
                SHIM._run_recall_route(Path("/bin/rekal"), Path("/repo"), "q", 5, {})


if __name__ == "__main__":
    unittest.main()
