#!/usr/bin/env python3
"""Create a missing remote release tag without replacing any existing ref."""

import os
import subprocess
import sys

from check_release_ref import VERSION, git, merged


def ensure_tag(tag, sha):
    if not VERSION.fullmatch(tag):
        raise ValueError("expected a stable version tag")
    if git("rev-parse", "HEAD") != sha:
        raise ValueError("checkout differs from validated release commit")
    merged(sha)
    ref = "refs/tags/" + tag
    entries = dict(line.split("\t")[::-1] for line in git("ls-remote", "origin", ref, ref + "^{}").splitlines())
    if ref in entries:
        if entries.get(ref + "^{}", entries[ref]) != sha:
            raise ValueError("remote tag points to another commit; refusing to replace it")
        return
    # No force: a concurrently created tag causes failure instead of replacement.
    subprocess.run(["git", "push", "origin", sha + ":" + ref], check=True)


if __name__ == "__main__":
    try:
        ensure_tag(os.environ.get("RELEASE_TAG", ""), os.environ.get("RELEASE_SHA", ""))
    except (ValueError, subprocess.CalledProcessError) as error:
        print(f"release: {error}", file=sys.stderr)
        sys.exit(1)
