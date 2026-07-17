#!/usr/bin/env python3
"""Seeded conversation-level dev/test split for industry-bench (WS-D).

Usage:
  python3 split_dataset.py --input data/longmemeval-s-conversations.jsonl \\
      --dev-ratio 0.2 --seed 42 --out-dir calibration/

Writes:
  calibration/split-<name>-dev.jsonl   (conversation ids, one per line)
  calibration/split-<name>-test.jsonl
  calibration/split-<name>-dev-conversations.jsonl
  calibration/split-<name>-test-conversations.jsonl
"""

from __future__ import annotations

import argparse
import json
import random
from pathlib import Path


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--input", type=Path, required=True)
    ap.add_argument("--name", default=None, help="dataset label (default: input stem)")
    ap.add_argument("--dev-ratio", type=float, default=0.2)
    ap.add_argument("--seed", type=int, default=42)
    ap.add_argument("--out-dir", type=Path, required=True)
    args = ap.parse_args()

    name = args.name or args.input.stem.replace("-conversations", "")
    conversations = []
    with open(args.input) as f:
        for line in f:
            if line.strip():
                conversations.append(json.loads(line))

    rng = random.Random(args.seed)
    ids = sorted(c["conversation_id"] for c in conversations)
    rng.shuffle(ids)
    n_dev = max(1, int(len(ids) * args.dev_ratio))
    dev_ids = set(ids[:n_dev])
    test_ids = set(ids[n_dev:])

    args.out_dir.mkdir(parents=True, exist_ok=True)
    id_dev = args.out_dir / f"split-{name}-dev-ids.txt"
    id_test = args.out_dir / f"split-{name}-test-ids.txt"
    id_dev.write_text("\n".join(sorted(dev_ids)) + "\n")
    id_test.write_text("\n".join(sorted(test_ids)) + "\n")

    dev_path = args.out_dir / f"split-{name}-dev-conversations.jsonl"
    test_path = args.out_dir / f"split-{name}-test-conversations.jsonl"
    with open(dev_path, "w") as fd, open(test_path, "w") as ft:
        for c in conversations:
            line = json.dumps(c, ensure_ascii=False) + "\n"
            if c["conversation_id"] in dev_ids:
                fd.write(line)
            else:
                ft.write(line)

    meta = {
        "dataset": name,
        "seed": args.seed,
        "dev_ratio": args.dev_ratio,
        "n_conversations": len(conversations),
        "n_dev": len(dev_ids),
        "n_test": len(test_ids),
        "input": str(args.input),
    }
    (args.out_dir / f"split-{name}-meta.json").write_text(json.dumps(meta, indent=2) + "\n")
    print(json.dumps(meta, indent=2))


if __name__ == "__main__":
    main()
