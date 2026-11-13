#!/usr/bin/env python3
"""Copy the exact current Rekal writer into an isolated, dependency-free Go harness."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parent
EXPECTED = "16cc899274af026f8a8d6cf8216d391be9aa8489"


def extract(text, name):
    start = text.index("func " + name + "(")
    end = text.index("\n}\n", start) + 3
    return text[start:end]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("repo", type=Path)
    args = parser.parse_args()
    commit = subprocess.check_output(["git", "-C", str(args.repo), "rev-parse", "HEAD"], text=True).strip()
    if commit != EXPECTED:
        raise SystemExit(f"Expected {EXPECTED}, got {commit}")
    exports = (args.repo / "cmd/rekal/cli/transport/export.go").read_text()
    gitx = (args.repo / "cmd/rekal/cli/gitx/git.go").read_text()
    writer = extract(exports, "CommitWireFormat")
    helper_names = ["ConfigValue", "HashObject", "RekalBranchName"]
    helpers = "\n".join(extract(gitx, name) for name in helper_names)
    harness = ROOT / "harness"
    (harness / "gitx").mkdir(parents=True, exist_ok=True)
    (harness / "transport").mkdir(exist_ok=True)
    (harness / "go.mod").write_text("module github.com/rekal-dev/rekal-cli/cmd/rekal/cli\n\ngo 1.25.6\n")
    (harness / "gitx/git.go").write_text('package gitx\n\nimport ("os/exec"; "strings")\n\n' + helpers)
    (harness / "transport/legacy.go").write_text('package transport\n\nimport ("fmt"; "os/exec"; "strings"; "github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx")\n\n' + writer)
    (ROOT / "source_provenance.json").write_text(json.dumps({
        "upstream_commit": commit,
        "upstream_file": "cmd/rekal/cli/transport/export.go",
        "function": "CommitWireFormat",
        "function_sha256": hashlib.sha256(writer.encode()).hexdigest(),
        "execution": "Exact function and its three Git helpers, compiled in isolation; not the full CLI or database pipeline.",
    }, indent=2) + "\n")
    for filename in ["LICENSE", "LICENSE.md", "LICENSE.txt"]:
        source = args.repo / filename
        if source.is_file():
            (ROOT / "UPSTREAM_LICENSE").write_bytes(source.read_bytes())
            break


if __name__ == "__main__":
    main()
