#!/usr/bin/env python3
"""Resolve a stable tag or an explicit main dispatch to one immutable commit."""

import os
import re
import subprocess
import sys


VERSION = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")


def git(*args):
    return subprocess.check_output(["git", *args], text=True).strip()


def merged(sha):
    git("rev-parse", "--verify", "origin/main")
    if subprocess.run(["git", "merge-base", "--is-ancestor", sha, "origin/main"], check=False).returncode:
        raise ValueError("release commit must be merged into origin/main")


def check_release_ref(ref):
    tag = ref.removeprefix("refs/tags/")
    if not ref.startswith("refs/tags/") or not VERSION.fullmatch(tag):
        raise ValueError("select an existing stable version tag (vMAJOR.MINOR.PATCH), not a branch")
    tagged = git("rev-parse", "--verify", ref + "^{commit}")
    if tagged != git("rev-parse", "HEAD"):
        raise ValueError("checkout does not match the selected version tag")
    merged(tagged)
    return tag


def resolve_release(ref, event, version, expected_sha):
    head = git("rev-parse", "HEAD")
    if expected_sha and expected_sha != head:
        raise ValueError("checkout does not match the workflow commit")
    if not version:
        return check_release_ref(ref), head
    if event != "workflow_dispatch" or ref != "refs/heads/main":
        raise ValueError("a new version requires workflow_dispatch on main")
    match = VERSION.fullmatch(version)
    if not match:
        raise ValueError("version must be vMAJOR.MINOR.PATCH without leading zeros or suffixes")
    merged(head)
    tags = git("tag", "--list").splitlines()
    if version in tags:
        if git("rev-parse", "refs/tags/" + version + "^{commit}") != head:
            raise ValueError("version tag already points to a different commit; never move it")
    else:
        numbered = [tuple(map(int, m.groups())) for tag in tags if (m := VERSION.fullmatch(tag))]
        if numbered and tuple(map(int, match.groups())) <= max(numbered):
            raise ValueError("new version must be greater than existing stable tags")
    return version, head


if __name__ == "__main__":
    try:
        tag, sha = resolve_release(
            os.environ.get("GITHUB_REF", ""), os.environ.get("GITHUB_EVENT_NAME", ""),
            os.environ.get("RELEASE_VERSION", ""), os.environ.get("GITHUB_SHA", ""),
        )
        if output := os.environ.get("GITHUB_OUTPUT"):
            with open(output, "a", encoding="utf-8") as stream:
                stream.write(f"tag={tag}\nsha={sha}\n")
        print(tag)
    except (ValueError, subprocess.CalledProcessError) as error:
        print(f"release: {error}", file=sys.stderr)
        sys.exit(1)
