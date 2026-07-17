#!/usr/bin/env python3
"""Aggregate industry-bench smoke manifests into one summary table."""

from __future__ import annotations

import argparse
import json
from pathlib import Path


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--smoke-dir", type=Path, required=True)
    ap.add_argument("--out", type=Path, required=True)
    args = ap.parse_args()

    rows = []
    for manifest_path in sorted(args.smoke_dir.glob("*/manifest.json")):
        m = json.loads(manifest_path.read_text())
        name = manifest_path.parent.name
        rows.append(
            {
                "run": name,
                "conversation_id": m.get("conversation_id"),
                "system": m.get("system"),
                "calibration": m.get("calibration"),
                "n_questions": m.get("n_questions"),
                "evidence_at_5_rate": m.get("evidence_at_5_rate"),
                "evidence_at_10_rate": m.get("evidence_at_10_rate"),
                "retrieved_context_tokens_mean": m.get("retrieved_context_tokens_mean"),
                "miss_reasons": m.get("miss_reasons"),
            }
        )

    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps({"runs": rows}, indent=2) + "\n")

    lines = [
        "# industry-bench smoke aggregate",
        "",
        f"- runs: {len(rows)}",
        "",
        "| run | n | ev@5 | ev@10 | ctx_tok |",
        "|---|---:|---:|---:|---:|",
    ]
    for r in rows:
        lines.append(
            f"| {r['run']} | {r.get('n_questions') or 0} | "
            f"{(r.get('evidence_at_5_rate') or 0):.2f} | "
            f"{(r.get('evidence_at_10_rate') or 0):.2f} | "
            f"{(r.get('retrieved_context_tokens_mean') or 0):.1f} |"
        )
    md = args.out.with_suffix(".md")
    md.write_text("\n".join(lines) + "\n")
    print(json.dumps({"runs": len(rows), "out": str(args.out)}, indent=2))


if __name__ == "__main__":
    main()
