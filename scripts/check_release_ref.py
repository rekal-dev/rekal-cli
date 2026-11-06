#!/usr/bin/env python3
"""Reject release dispatches that do not identify a merged stable version tag."""

import os
import re
import subprocess
import sys


def git(*args):
    return subprocess.check_output(["git", *args], text=True).strip()


def check_release_ref(ref):
    if not re.fullmatch(r"refs/tags/v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", ref):
        raise ValueError("select an existing stable version tag (vMAJOR.MINOR.PATCH), not a branch")
    tagged = git("rev-parse", "--verify", ref + "^{commit}")
    if tagged != git("rev-parse", "HEAD"):
        raise ValueError("checkout does not match the selected version tag")
    git("rev-parse", "--verify", "origin/main")
    if subprocess.run(["git", "merge-base", "--is-ancestor", tagged, "origin/main"], check=False).returncode:
        raise ValueError("release tag must point to a commit merged into origin/main")
    return ref.removeprefix("refs/tags/")


if __name__ == "__main__":
    try:
        print(check_release_ref(os.environ.get("GITHUB_REF", "")))
    except (ValueError, subprocess.CalledProcessError) as error:
        print(f"release: {error}", file=sys.stderr)
        sys.exit(1)
