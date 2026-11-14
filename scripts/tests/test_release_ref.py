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

    def check(self, ref, **extra):
        return subprocess.run(
            [sys.executable, str(GUARD)], cwd=self.root,
            env={**self.env, "GITHUB_REF": ref, "GITHUB_EVENT_NAME": "push",
                 "RELEASE_VERSION": "", "GITHUB_SHA": "", "GITHUB_OUTPUT": "", **extra}, capture_output=True, text=True, check=False,
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

    def test_main_dispatch_pins_sha_without_creating_tag(self):
        sha = self.git("rev-parse", "HEAD")
        output = self.root / "outputs"
        result = self.check("refs/heads/main", GITHUB_EVENT_NAME="workflow_dispatch",
                            RELEASE_VERSION="v2.0.0", GITHUB_SHA=sha, GITHUB_OUTPUT=str(output))
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(output.read_text(), f"tag=v2.0.0\nsha={sha}\n")
        self.assertEqual(self.git("tag", "--list"), "")

    def test_new_version_rejects_wrong_event_branch_sha_and_input(self):
        sha = self.git("rev-parse", "HEAD")
        cases = [
            ("refs/heads/feature", "workflow_dispatch", "v2.0.0", sha),
            ("refs/heads/main", "push", "v2.0.0", sha),
            ("refs/heads/main", "workflow_dispatch", "v2.0.0", "0" * 40),
            ("refs/heads/main", "workflow_dispatch", "v2.0.0; echo unsafe", sha),
            ("refs/heads/main", "workflow_dispatch", "v2.0.0\nsha=bad", sha),
            ("refs/heads/main", "workflow_dispatch", "v02.0.0", sha),
        ]
        for ref, event, version, expected in cases:
            with self.subTest(ref=ref, version=version):
                self.assertNotEqual(self.check(ref, GITHUB_EVENT_NAME=event,
                    RELEASE_VERSION=version, GITHUB_SHA=expected).returncode, 0)

    def test_main_dispatch_rejects_unmerged_checkout(self):
        self.commit("not on origin main")
        result = self.check("refs/heads/main", GITHUB_EVENT_NAME="workflow_dispatch", RELEASE_VERSION="v2.0.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("merged into origin/main", result.stderr)

    def test_retry_same_tag_but_never_move_or_downgrade(self):
        self.git("tag", "v2.0.0")
        self.assertEqual(self.check("refs/heads/main", GITHUB_EVENT_NAME="workflow_dispatch",
                                   RELEASE_VERSION="v2.0.0").returncode, 0)
        self.assertNotEqual(self.check("refs/heads/main", GITHUB_EVENT_NAME="workflow_dispatch",
                                      RELEASE_VERSION="v1.9.0").returncode, 0)
        self.commit("new main")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")
        result = self.check("refs/heads/main", GITHUB_EVENT_NAME="workflow_dispatch", RELEASE_VERSION="v2.0.0")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("never move it", result.stderr)

    def test_remote_tag_creation_retry_and_collision(self):
        remote = self.root / "remote.git"
        self.git("init", "--bare", "-q", str(remote))
        self.git("remote", "add", "origin", str(remote))
        script = GUARD.with_name("ensure_release_tag.py")
        sha = self.git("rev-parse", "HEAD")
        def publish(version="v2.0.0"):
            return subprocess.run([sys.executable, str(script)], cwd=self.root,
                env={**self.env, "RELEASE_TAG": version, "RELEASE_SHA": self.git("rev-parse", "HEAD")},
                capture_output=True, text=True, check=False)
        result = publish()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(sha, self.git("ls-remote", "origin", "refs/tags/v2.0.0"))
        self.assertEqual(publish().returncode, 0)
        self.git("-c", "tag.gpgsign=false", "tag", "-a", "v2.0.1", "-m", "annotated")
        self.git("push", "origin", "refs/tags/v2.0.1")
        self.assertEqual(publish("v2.0.1").returncode, 0)
        self.commit("different commit")
        self.git("update-ref", "refs/remotes/origin/main", "HEAD")
        result = publish()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("refusing to replace", result.stderr)
        self.assertIn(sha, self.git("ls-remote", "origin", "refs/tags/v2.0.0"))


if __name__ == "__main__":
    unittest.main()
