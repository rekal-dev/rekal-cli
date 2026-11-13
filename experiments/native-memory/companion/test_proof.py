#!/usr/bin/env python3
"""Executable tests: real Git + exact upstream Go writer + experimental archive."""
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from memory_proof import MAGIC, canonical, capture, decode, digest, rebuild, validated_state

ROOT = Path(__file__).resolve().parent
PROBE = ROOT / "probe"
SESSION = "01J" + "0" * 23
SOURCE = {"adapter": "claude", "profile": "synthetic-profile", "repo": "demo-repo",
          "owner": "alice", "scope": "repo_shared", "root": "memory"}
NATIVE_TIME = "2026-04-02T10:00:00Z"
OBSERVED = "2026-11-07T12:00:00Z"
NOTE = (f"---\nmodified: {NATIVE_TIME}\n---\nUse idempotency keys for payments.\n"
        f"Source: {SESSION}:t42:bytes0-30\n").encode()
SHARED_REF = "refs/heads/rekal/alice@example.invalid"
BODY0 = b"RKLBODY\x01\x01"
DICT0 = b"RKDICT\x01\x00\x00\x00\x00\x00"


def archive():
    return capture(MAGIC, SOURCE, {"payments.md": NOTE}, OBSERVED, "1" * 40)


class ArchiveProof(unittest.TestCase):
    def test_onboarding_preserves_existing_note_and_distinct_timestamps(self):
        data = archive()
        rows = rebuild(data)["memories"]
        self.assertEqual(rows[0]["content"].encode(), NOTE)
        self.assertEqual(rows[0]["source_modified_at"], NATIVE_TIME)
        self.assertEqual(rows[0]["observed_at"], OBSERVED)
        self.assertEqual(rows[0]["observed_at_commit"], "1" * 40)

    def test_identical_onboarding_and_unchanged_snapshot_are_noops(self):
        data = archive()
        self.assertEqual(capture(data, SOURCE, {"payments.md": NOTE}, "2026-11-08T00:00:00Z", "2" * 40), data)

    def test_memory_only_change_appends_preserving_previous_bytes(self):
        before = archive()
        changed = NOTE + b"Retry only transient failures.\n"
        after = capture(before, SOURCE, {"payments.md": changed}, "2026-11-08T00:00:00Z", "1" * 40)
        self.assertTrue(after.startswith(before))
        self.assertEqual(len(decode(after)), 2)
        self.assertEqual(rebuild(after)["memories"][0]["content"].encode(), changed)
        self.assertEqual(validated_state(after)[1][digest(NOTE)], NOTE)

    def test_unchanged_content_is_stored_once_across_snapshots_and_sources(self):
        before = archive()
        after = capture(before, SOURCE, {"payments.md": NOTE, "build.md": b"Use make check.\n"}, OBSERVED, "1" * 40)
        self.assertEqual(len(decode(after)[-1]["new_blobs"]), 1)
        bob = {**SOURCE, "owner": "bob"}
        after = capture(after, bob, {"payments.md": NOTE}, OBSERVED, "1" * 40)
        self.assertEqual(decode(after)[-1]["new_blobs"], {})
        self.assertEqual(len(validated_state(after)[1]), 2)

    def test_unknown_native_time_stays_unknown(self):
        data = capture(MAGIC, SOURCE, {"old.md": b"A pre-existing note with no date.\n"}, OBSERVED, "1" * 40)
        self.assertIsNone(rebuild(data)["memories"][0]["source_modified_at"])

    def test_delete_and_rebuild_index_gives_identical_ids_and_backlinks(self):
        data = archive()
        with tempfile.TemporaryDirectory() as temp:
            index = Path(temp) / "index.json"
            index.write_bytes(canonical(rebuild(data)))
            first = index.read_bytes()
            index.unlink()
            index.write_bytes(canonical(rebuild(data)))
            self.assertEqual(index.read_bytes(), first)
        link = rebuild(data)["backlinks"][0]
        self.assertEqual((link["session_id"], link["turn"], link["byte_start"], link["byte_end"]), (SESSION, 42, 0, 30))
        self.assertEqual(link["relationship"], "explicit_citation_not_verified_support")

    def test_no_citation_does_not_invent_a_session_link(self):
        data = capture(MAGIC, SOURCE, {"note.md": b"Observed at a commit, but no source citation.\n"}, OBSERVED, "1" * 40)
        self.assertEqual(rebuild(data)["backlinks"], [])

    def test_personal_and_repo_private_sources_are_excluded_before_serialization(self):
        data = archive()
        for scope in ("personal", "account", "repo_private", "agent_private"):
            with self.subTest(scope=scope):
                with self.assertRaises(PermissionError):
                    capture(data, {**SOURCE, "scope": scope}, {"private.md": b"PRIVATE_FIXTURE_NEVER_EXPORT"}, OBSERVED, "1" * 40)
        self.assertNotIn(b"PRIVATE_FIXTURE_NEVER_EXPORT", canonical(rebuild(data)))
        self.assertEqual(data, archive())

    def test_same_path_different_owners_and_profiles_remains_distinct(self):
        data = archive()
        for override in ({"owner": "bob"}, {"profile": "second-profile"}, {"repo": "second-repo"}):
            data = capture(data, {**SOURCE, **override}, {"payments.md": NOTE}, OBSERVED, "1" * 40)
        rows = rebuild(data)["memories"]
        self.assertEqual(len(rows), 4)
        self.assertEqual(len({row["id"] for row in rows}), 4)

    def test_removal_is_source_local_and_retains_old_snapshot(self):
        data = archive()
        data = capture(data, {**SOURCE, "owner": "bob"}, {"payments.md": NOTE}, OBSERVED, "1" * 40)
        data = capture(data, SOURCE, {}, "2026-11-08T00:00:00Z", "1" * 40)
        rows = rebuild(data)["memories"]
        self.assertEqual(len(rows), 1)
        self.assertEqual(rows[0]["source"]["owner"], "bob")
        self.assertEqual(validated_state(data)[1][digest(NOTE)], NOTE)

    def test_incomplete_scan_cannot_delete_source_membership(self):
        data = archive()
        with self.assertRaises(ValueError):
            capture(data, SOURCE, {}, OBSERVED, "1" * 40, complete=False)
        self.assertEqual(len(rebuild(data)["memories"]), 1)

    def test_truncation_corruption_and_unsupported_version_are_rejected(self):
        data = archive()
        cases = [data[:-1], data[:-20], data[:10], b"RKMEM\x02" + data[len(MAGIC):]]
        corrupt = bytearray(data)
        corrupt[len(MAGIC)+4] ^= 1
        cases.append(bytes(corrupt))
        for invalid in cases:
            with self.subTest(length=len(invalid)):
                with self.assertRaises(ValueError):
                    validated_state(invalid)


class GitProof(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="rekal-memory-proof-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.repo = self.root / "author"
        self.repo.mkdir()
        self.env = dict(os.environ)
        for key in list(self.env):
            if key.startswith("GIT_"):
                del self.env[key]
        self.env.update(GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull)
        self.git("init", "--initial-branch=main")
        self.git("config", "user.name", "Synthetic Alice")
        self.git("config", "user.email", "alice@example.invalid")
        self.git("config", "core.hooksPath", os.devnull)
        (self.repo / "README.md").write_text("Synthetic local proof repository.\n")
        self.git("add", "README.md")
        self.git("commit", "-m", "synthetic initial commit")
        self.code_commit = self.git("rev-parse", "HEAD").strip().decode()
        self.memory = capture(MAGIC, SOURCE, {"payments.md": NOTE}, OBSERVED, self.code_commit)
        self.initial = self.seed_ref(SHARED_REF, {"rekal.body": BODY0, "dict.bin": DICT0, "memory.bin": self.memory})

    def git(self, *args, repo=None, data=None, check=True):
        result = subprocess.run(["git", "-C", str(repo or self.repo), *args], input=data,
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=self.env)
        if check and result.returncode:
            raise AssertionError(result.stderr.decode())
        return result.stdout

    def seed_ref(self, ref, files):
        entries = []
        for path, data in sorted(files.items()):
            blob = self.git("hash-object", "-w", "--stdin", data=data).strip().decode()
            entries.append(f"100644 blob {blob}\t{path}\n")
        tree = self.git("mktree", data="".join(entries).encode()).strip().decode()
        commit = self.git("commit-tree", tree, "-m", "synthetic wire baseline").strip().decode()
        self.git("update-ref", ref, commit)
        return commit

    def write(self, mode, updates=None, ref=SHARED_REF, parent="-", success=True):
        updates = updates or {}
        paths = {}
        for index, (name, data) in enumerate(sorted(updates.items())):
            path = self.root / f"input-{index}.bin"
            path.write_bytes(data)
            paths[name] = str(path)
        if mode == "legacy":
            args = [str(PROBE), mode, str(self.repo), paths["rekal.body"], paths["dict.bin"]]
        else:
            args = [str(PROBE), mode, str(self.repo), ref, parent]
            for name, path in paths.items():
                args.extend([name, path])
        result = subprocess.run(args, cwd=self.repo, env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        if success:
            self.assertEqual(result.returncode, 0, result.stderr.decode())
            return result.stdout.strip().decode()
        self.assertNotEqual(result.returncode, 0)
        return result

    def show(self, path, ref=SHARED_REF, repo=None):
        return self.git("show", f"{ref}:{path}", repo=repo)

    def test_exact_current_writer_omits_companion_from_new_tip_but_history_retains_it(self):
        commit = self.write("legacy", {"rekal.body": BODY0 + b"new fixture frame", "dict.bin": DICT0})
        names = self.git("ls-tree", "--name-only", commit).decode().splitlines()
        self.assertEqual(names, ["dict.bin", "rekal.body"])
        self.assertEqual(self.show("memory.bin", ref=self.initial), self.memory)
        self.assertEqual(self.git("rev-parse", f"{commit}^").strip().decode(), self.initial)

    def test_proposed_writer_preserves_memory_on_session_only_update(self):
        self.write("preserve", {"rekal.body": BODY0 + b"new fixture frame", "dict.bin": DICT0})
        self.assertEqual(self.show("memory.bin"), self.memory)
        self.assertEqual(self.show("rekal.body"), BODY0 + b"new fixture frame")

    def test_memory_only_update_preserves_both_session_files(self):
        updated = capture(self.memory, SOURCE, {"payments.md": NOTE + b"New memory-only edit.\n"}, OBSERVED, "1" * 40)
        self.write("preserve", {"memory.bin": updated})
        self.assertEqual(self.show("rekal.body"), BODY0)
        self.assertEqual(self.show("dict.bin"), DICT0)
        self.assertEqual(self.show("memory.bin"), updated)
        self.assertTrue(updated.startswith(self.memory))

    def test_local_remote_push_fresh_clone_and_index_rebuild(self):
        self.write("preserve", {"rekal.body": BODY0 + b"next", "dict.bin": DICT0})
        bare = self.root / "remote.git"
        self.git("init", "--bare", "--initial-branch=main", str(bare))
        self.git("remote", "add", "origin", str(bare))
        self.git("push", "origin", "main", SHARED_REF)
        teammate = self.root / "teammate"
        self.git("clone", str(bare), str(teammate))
        imported = self.show("memory.bin", ref="refs/remotes/origin/rekal/alice@example.invalid", repo=teammate)
        self.assertEqual(imported, self.memory)
        self.assertEqual(canonical(rebuild(imported)), canonical(rebuild(self.memory)))
        captured_commit = rebuild(imported)["memories"][0]["observed_at_commit"]
        self.git("cat-file", "-e", captured_commit + "^{commit}", repo=teammate)
        self.assertFalse((teammate / "memory.bin").exists())

    def test_older_writer_still_omits_companion_after_a_new_writer(self):
        updated = capture(self.memory, SOURCE, {"payments.md": NOTE + b"Newer content.\n"}, OBSERVED, "1" * 40)
        modern = self.write("preserve", {"memory.bin": updated})
        self.write("legacy", {"rekal.body": BODY0 + b"old client push", "dict.bin": DICT0})
        names = self.git("ls-tree", "--name-only", SHARED_REF).decode().splitlines()
        self.assertNotIn("memory.bin", names)
        self.assertEqual(self.show("memory.bin", ref=modern), updated)

    def test_independent_memory_branch_survives_unchanged_old_writer(self):
        memory_ref = "refs/heads/rekal-memory/alice-example"
        self.seed_ref(memory_ref, {"memory.bin": self.memory})
        self.write("legacy", {"rekal.body": BODY0 + b"old client", "dict.bin": DICT0})
        self.assertEqual(self.show("memory.bin", ref=memory_ref), self.memory)

    def test_stale_concurrent_writer_is_rejected_without_lost_update(self):
        updated = capture(self.memory, SOURCE, {"payments.md": NOTE + b"Writer one.\n"}, OBSERVED, "1" * 40)
        winner = self.write("preserve", {"memory.bin": updated}, parent=self.initial)
        self.write("preserve", {"rekal.body": BODY0 + b"stale writer"}, parent=self.initial, success=False)
        self.assertEqual(self.git("rev-parse", SHARED_REF).strip().decode(), winner)
        self.assertEqual(self.show("memory.bin"), updated)
        self.assertEqual(self.show("rekal.body"), BODY0)

    def test_user_worktree_index_and_unrelated_sidecar_are_preserved(self):
        self.seed_ref(SHARED_REF, {"rekal.body": BODY0, "dict.bin": DICT0, "memory.bin": self.memory, "future.bin": b"unknown future data"})
        (self.repo / "staged.txt").write_text("Keep my staged changes.\n")
        self.git("add", "staged.txt")
        before = self.git("ls-files", "--stage")
        self.write("preserve", {"memory.bin": self.memory})
        self.assertEqual(self.git("ls-files", "--stage"), before)
        self.assertEqual(self.show("future.bin"), b"unknown future data")


class RecordingResult(unittest.TextTestResult):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.cases = []

    def addSuccess(self, test):
        super().addSuccess(test)
        self.cases.append({"test": test.id(), "status": "passed"})

    def addFailure(self, test, err):
        super().addFailure(test, err)
        self.cases.append({"test": test.id(), "status": "failed"})

    def addError(self, test, err):
        super().addError(test, err)
        self.cases.append({"test": test.id(), "status": "error"})


if __name__ == "__main__":
    if not PROBE.is_file():
        raise SystemExit("Compile harness first: cd harness && go build -o ../probe ./cmd/probe")
    stream = io.StringIO()
    suite = unittest.defaultTestLoader.loadTestsFromModule(__import__(__name__))
    result = unittest.TextTestRunner(stream=stream, verbosity=2, resultclass=RecordingResult).run(suite)
    print(stream.getvalue(), end="")
    (ROOT / "test-results.txt").write_text(stream.getvalue())
    demo = archive()
    (ROOT / "example-memory.bin").write_bytes(demo)
    (ROOT / "example-index.json").write_bytes(canonical(rebuild(demo)) + b"\n")
    (ROOT / "results.json").write_text(json.dumps({
        "tests_run": result.testsRun, "passed": result.wasSuccessful(), "cases": result.cases,
        "archive_bytes": len(demo), "synthetic_note_bytes": len(NOTE),
        "archive_sha256": digest(demo), "rebuilt_index_sha256": digest(canonical(rebuild(demo))),
        "scope": "Synthetic contract and Git-writer proof; not integrated Rekal CLI, retrieval benchmark, or production format.",
    }, indent=2) + "\n")
    raise SystemExit(0 if result.wasSuccessful() else 1)
