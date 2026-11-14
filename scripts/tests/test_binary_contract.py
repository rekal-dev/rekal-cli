"""Subprocess contracts. Run with REKAL_TEST_BINARY=/absolute/path/to/rekal.

All homes, agent transcripts, repositories and remotes are disposable synthetic
fixtures. These tests never read the developer's native agent memory.
"""
import base64
import http.server
import threading
import datetime
import json
import os
import re
from pathlib import Path
import subprocess
import tempfile
import unittest


@unittest.skipUnless(os.environ.get("REKAL_TEST_BINARY"), "set REKAL_TEST_BINARY to test the executable")
class BinaryContract(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="rekal-binary-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        self.home = self.root / "home"
        self.home.mkdir()
        self.binary = str(Path(os.environ["REKAL_TEST_BINARY"]).resolve())
        self.env = dict(os.environ)
        for key in ("CLAUDE_CODE_PROJECT_DIR_NAME", "REKAL_SKIP_CHECKPOINT", "REKAL_BENCH"):
            self.env.pop(key, None)
        for key in ("HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME",
                    "CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME",
                    "GEMINI_CLI_HOME", "OPENCODE_CONFIG_DIR", "REKAL_CONFIG_HOME"):
            self.env[key] = str(self.home / key.lower())
            Path(self.env[key]).mkdir()
        self.env.update(GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_SYSTEM=os.devnull)
        self.git("init", "-b", "main")
        self.git("config", "user.name", "Synthetic")
        self.git("config", "user.email", "synthetic@example.invalid")
        self.git("config", "core.hooksPath", "/dev/null")
        self.git("commit", "--allow-empty", "-m", "synthetic baseline")
        self.run_cli("init")
        encoded = re.sub(r"[^a-zA-Z0-9]", "-", str(self.repo))
        self.project = Path(self.env["CLAUDE_CONFIG_DIR"]) / "projects" / encoded
        self.project.mkdir(parents=True, exist_ok=True)
        self.transcript = self.project / "synthetic.jsonl"
        self.notes = self.project / "memory"
        self.notes.mkdir()
        (self.notes / "decision.md").write_text("quartzdecision repository note: keep evidence.\n")
        self.texts = [f"quartzdecision turn {i}" for i in range(25)]
        self.capture()

    def command(self, argv, cwd=None):
        return subprocess.run(argv, cwd=cwd or self.repo, env=self.env,
                              text=True, capture_output=True, timeout=60)

    def git(self, *args):
        p = self.command(["git", *args])
        self.assertEqual(p.returncode, 0, p.stderr)
        return p.stdout.strip()

    def run_cli(self, *args, ok=True, cwd=None):
        p = self.command([self.binary, *args], cwd)
        if ok:
            self.assertEqual(p.returncode, 0, f"{args}: {p.stderr}\n{p.stdout}")
        else:
            self.assertNotEqual(p.returncode, 0, f"accepted invalid input {args}")
            self.assertNotIn("panic:", p.stderr)
        return p

    def data(self, *args, cwd=None):
        return json.loads(self.run_cli(*args, "--json", cwd=cwd).stdout)

    def capture(self):
        lines = [dict(type="summary", sessionId="synthetic-contract")]
        for i, content in enumerate(self.texts):
            ts = datetime.datetime(2026, 1, 1, tzinfo=datetime.timezone.utc) + datetime.timedelta(seconds=i)
            lines.append(dict(type="user", sessionId="synthetic-contract", uuid=f"turn-{i}",
                              timestamp=ts.isoformat(), message=dict(role="user", content=content)))
        self.transcript.write_text("".join(json.dumps(row) + "\n" for row in lines))
        self.run_cli("checkpoint")

    def read_all(self, ref):
        pages = [self.data("query", "--ref", ref)]
        while pages[-1].get("next"):
            self.assertLess(len(pages), 100, "continuation loop")
            pages.append(self.data("query", "--cursor", pages[-1]["next"]))
        return pages

    def test_search_read_reindex_and_filters(self):
        out = self.data("quartzdecision", "--limit", "3")
        self.assertEqual(out["literal"]["total"], 25)
        self.assertEqual(len(out["literal"]["hits"]), 3)
        hits = list(out["literal"]["hits"])
        page = out["literal"]
        while page.get("next"):
            page = self.data("--cursor", page["next"])["literal"]
            hits.extend(page["hits"])
        self.assertEqual(len({h["ref"] for h in hits}), 25)
        ref = hits[13]["ref"]
        read = self.data("query", "--ref", ref)
        self.assertEqual(read["evidence"]["windows"][0]["turns"][0]["content"], self.texts[13])
        alias = self.data("query", "--ref", "s1")["ref"]
        self.run_cli("index")
        self.assertEqual(self.data("query", "--ref", ref)["evidence"], read["evidence"])
        self.assertEqual(self.data("query", "--ref", alias)["ref"], alias)
        self.assertEqual(self.data("quartzdecision", "--author", "nobody")["literal"]["total"], 0)
        self.assertEqual(self.data("quartzdecision", "--actor", "agent")["literal"]["total"], 0)
        self.assertEqual(self.data("quartzdecision", "--commit", self.git("rev-parse", "HEAD")[:8])["literal"]["total"], 25)
        self.assertEqual(self.data("quartzdecision", "--file", "no-such-file")["literal"]["total"], 0)
        zero = self.data("quartzdecision", "--limit", "0")["literal"]
        self.assertEqual(zero["hits"], [])
        self.assertNotIn("next", zero)
        self.assertEqual(self.data("no-such-literal-xyz")["literal"]["total"], 0)

    def test_cursor_invalidated_and_argument_errors(self):
        cursor = self.data("quartzdecision", "--limit", "2")["literal"]["next"]
        self.texts.append("quartzdecision appended")
        self.capture()
        self.assertIn("changed", self.run_cli("--cursor", cursor, ok=False).stderr)
        empty = base64.urlsafe_b64encode(json.dumps(dict(v=1, q="", n=2, s="x")).encode()).decode().rstrip("=")
        for args in [("--cursor", empty), ("--cursor", "e30"), ("--cursor", "bad!"),
                     ("--cursor", cursor, "extra"), ("--cursor", cursor, "--limit", "2"),
                     ("query", "--cursor", "bad!"), ("query", "--ref", "memory:missing"),
                     ("query", "--ref", "s1", "--sql", "SELECT 1"),
                     ("query", "--ref", "s1", "--cursor", "e30"),
                     ("query", "--ref", "s1", "--limit", "3"),
                     ("query", "--ref", "passage:s1:2-1"),
                     ("quartzdecision", "--limit", "-1"), ("quartzdecision", "--actor", "typo")]:
            with self.subTest(args=args):
                self.run_cli(*args, ok=False)

    def test_long_unicode_reads_and_literal_location(self):
        # Exercise text/reference contracts independently of native-model CPU
        # throughput. A real Nomic retrieval run lives in scripts/passage-eval.
        # The HTTP endpoint is local and receives only this synthetic fixture.
        requests = []
        class Embeddings(http.server.BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass

            def do_POST(self):
                payload = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
                requests.append(len(payload["input"]))
                response = json.dumps({"data": [
                    {"index": i, "embedding": [1.0, 0.0, 0.0, 0.0]}
                    for i, _ in enumerate(payload["input"])
                ]}).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(response)))
                self.end_headers()
                self.wfile.write(response)

        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Embeddings)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        def stop_server():
            server.shutdown()
            server.server_close()
            thread.join(timeout=5)
        self.addCleanup(stop_server)
        config = {"embedding": {"endpoint": f"http://127.0.0.1:{server.server_port}/v1",
                                "model": "binary-contract-fixture", "timeout_seconds": 5}}
        (self.repo / ".rekal" / "config.json").write_text(json.dumps(config))
        self.texts = [f"item {i}" for i in range(300)]
        self.texts[289] = "retry " + "旧背景 " * 5000 + "retry final decision"
        self.capture()
        out = self.data("retry final decision", "--limit", "1")
        hit = out["literal"]["hits"][0]
        self.assertIn("@", hit["ref"])
        passage = self.read_all(hit["ref"])
        text = "".join(t["content"] for p in passage for w in p["evidence"]["windows"] for t in w["turns"])
        self.assertIn("retry final decision", text)
        pages = self.read_all("session:" + hit["session_id"])
        turns = [t for p in pages for w in p["evidence"]["windows"] for t in w["turns"]]
        reconstructed = {}
        for turn in turns:
            reconstructed[turn["index"]] = reconstructed.get(turn["index"], "") + turn["content"]
        self.assertEqual([reconstructed[i] for i in range(300)], self.texts)
        edges = [json.loads(line) for line in (self.repo / ".rekal" / "recall-log.ndjson").read_text().splitlines()]
        drills = [e for e in edges if e["kind"] == "drill" and e["target"] == hit["session_id"]]
        self.assertEqual(len(drills), 2, "passage + session read should log once each, not once per page")
        self.assertGreater(len(requests), 0, "fixture embedding endpoint was not exercised")

    def test_memory_history_local_expiry_and_no_export(self):
        note = self.notes / "decision.md"
        note.write_text("quartzdecision " + "界" * 12000)
        self.capture()
        out = self.data("quartzdecision")
        ref = next(m["ref"] for m in out["memories"] if m["ref"].startswith("memory:"))
        self.assertEqual("".join(p.get("content", "") for p in self.read_all(ref)), note.read_text())
        text_read = self.run_cli("query", "--ref", ref).stdout
        self.assertIn('owner="synthetic@example.invalid"', text_read)
        self.assertIn("commit=" + self.git("rev-parse", "HEAD"), text_read)
        note.unlink()
        self.capture()
        self.assertTrue(self.data("query", "--ref", ref)["content"].startswith("quartzdecision"))
        local = self.root / "personal"
        local.mkdir()
        personal = local / "note.md"
        personal.write_text("privatequartz local only")
        config = self.repo / ".rekal" / "config.json"
        config.write_text(json.dumps(dict(memory_sources=[dict(name="personal", adapter="markdown", scope="local", path=str(local))])))
        local_ref = self.data("privatequartz")["memories"][0]["ref"]
        self.assertTrue(local_ref.startswith("local-memory:"))
        self.assertEqual(self.data("query", "--ref", local_ref)["content"], personal.read_text())
        personal.write_text("privatequartz changed")
        self.run_cli("query", "--ref", local_ref, ok=False)
        new_ref = self.data("privatequartz")["memories"][0]["ref"]
        self.run_cli("index")
        self.assertEqual(self.data("privatequartz")["memories"][0]["ref"], new_ref)
        config.write_text('{"memory":false}')
        self.run_cli("query", "--ref", new_ref, ok=False)
        count = self.run_cli("query", "--sql", "SELECT count(*) AS n FROM memory_revisions WHERE payload LIKE '%privatequartz%'", "--json")
        self.assertEqual(json.loads(count.stdout)["n"], 0)

    def test_local_remote_roundtrip_and_append_only_body(self):
        bare = self.root / "remote.git"
        self.git("init", "--bare", "-b", "main", str(bare))
        self.git("remote", "add", "origin", str(bare))
        self.git("push", "--no-verify", "-u", "origin", "main")
        self.run_cli("push", "--strict")
        ref = next(m["ref"] for m in self.data("quartzdecision")["memories"]
                   if m["ref"].startswith("memory:"))
        branch = "rekal/synthetic@example.invalid"
        def body():
            p = subprocess.run(["git", "show", branch + ":rekal.body"],
                               cwd=self.repo, env=self.env, capture_output=True, timeout=30)
            self.assertEqual(p.returncode, 0, p.stderr)
            return p.stdout
        first = body()
        (self.notes / "decision.md").write_text("quartzdecision revised repository note")
        self.git("commit", "--allow-empty", "-m", "synthetic second observation")
        self.capture()
        self.git("push", "--no-verify", "origin", "main")
        self.run_cli("push", "--strict")
        self.assertTrue(body().startswith(first), "ordinary export rewrote the old body")
        peer = self.root / "peer"
        self.git("clone", str(bare), str(peer))
        for args in [("user.name", "Synthetic Peer"), ("user.email", "peer@example.invalid"),
                     ("core.hooksPath", "/dev/null")]:
            p = self.command(["git", "config", *args], cwd=peer)
            self.assertEqual(p.returncode, 0, p.stderr)
        self.run_cli("init", cwd=peer)
        self.run_cli("sync", cwd=peer)
        self.assertIn("keep evidence", self.data("query", "--ref", ref, cwd=peer)["content"])
        latest = self.data("quartzdecision", cwd=peer)
        self.assertTrue(any("revised repository note" in m["snippet"] for m in latest["memories"]))
        self.assertEqual(latest["literal"]["total"], 25)

    def test_help_and_legacy_reads(self):
        for command in ("", "query", "find", "evidence", "init", "clean", "checkpoint", "index", "embed", "sync", "push", "log", "version"):
            args = [command] if command else []
            self.assertTrue(self.run_cli(*args, "--help").stdout)
        self.assertTrue(self.run_cli("version").stdout.strip())
        self.assertIn("quartzdecision", self.run_cli("find", "quartzdecision").stdout)
        self.assertTrue(self.data("query", "--session", "s1", "--limit", "2")["turns"])
        self.assertEqual(json.loads(self.run_cli("query", "--sql", "SELECT 1 AS n", "--json").stdout)["n"], 1)
