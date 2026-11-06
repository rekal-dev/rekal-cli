"""Exercise release-ref checks against real local git histories, without a remote."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


GUARD = Path(__file__).resolve().parents[1] / "check_release_ref.py"


class ReleaseRefTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = {**os.environ, "GIT_CONFIG_GLOBAL": os.devnull, "GIT_CONFIG_SYSTEM": os.devnull}
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "release-test@example.com")
        self.git("config", "user.name", "Release Test")
        self.commit("initial")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.root, env=self.env, text=True).strip()

    def commit(self, text):
        (self.root / "source.txt").write_text(text)
        self.git("add", "source.txt")
        self.git("-c", "commit.gpgsign=false", "commit", "-q", "-m", text)

    def check(self, ref):
        return subprocess.run(
            [sys.executable, str(GUARD)], cwd=self.root,
            env={**self.env, "GITHUB_REF": ref}, capture_output=True, text=True, check=False,
        )

    def test_merged_lightweight_tag(self):
        self.git("tag", "v1.0.3")
        result = self.check("refs/tags/v1.0.3")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "v1.0.3")

    def test_merged_annotated_tag(self):
        self.git("-c", "tag.gpgsign=false", "tag", "-a", "v1.0.3", "-m", "release")
        result = self.check("refs/tags/v1.0.3")
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_branch_dispatch_and_invalid_versions_are_rejected(self):
        for ref in ["refs/heads/main", "refs/tags/main", "refs/tags/v01.0.3", "refs/tags/v1.3", ""]:
            with self.subTest(ref=ref):
                result = self.check(ref)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("existing stable version tag", result.stderr)

    def test_unmerged_tag_is_rejected(self):
        self.git("checkout", "-q", "-b", "feature")
        self.commit("unmerged work")
        self.git("tag", "v1.0.3")
        result = self.check("refs/tags/v1.0.3")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("merged into origin/main", result.stderr)

    def test_mismatched_checkout_is_rejected(self):
        self.git("tag", "v1.0.3")
        self.commit("new head")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")
        result = self.check("refs/tags/v1.0.3")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checkout does not match", result.stderr)


if __name__ == "__main__":
    unittest.main()
