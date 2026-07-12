#!/usr/bin/env python3
"""usage_mine.py <outdir> — observational Rekal-usage & effectiveness signals
mined from the ledger's own records (06-eval-strategy.md §4a). No new labels,
no LLM: pure SQL over the index DB, aggregates only.

Reports, across the whole indexed corpus:
  - adoption: sessions (and repos) that invoked `rekal` at all
  - intensity: rekal invocations per using-session
  - drill-through: using-sessions that followed a search with `rekal query
    --session` (an implicit relevance signal — did recall return something
    worth reading?)
  - steering delta: mean human_steering turns per session, rekal-using vs not
    (a natural experiment — confounded, but large-N and directional)

Writes usage.md (markdown) and usage.json (the aggregates). Stdlib only.
"""
import json
import pathlib
import subprocess
import sys


def q(sql: str) -> list[dict]:
    out = subprocess.run(
        ["rekal", "query", "--index", sql], capture_output=True, text=True, timeout=300
    )
    if out.returncode != 0:
        sys.exit(f"rekal query failed:\n{out.stderr}")
    rows = []
    for line in out.stdout.splitlines():
        line = line.strip()
        if line:
            rows.append(json.loads(line))
    return rows


def mean(xs: list[float]) -> float:
    return sum(xs) / len(xs) if xs else 0.0


def main() -> None:
    outdir = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".")

    # Per-session rekal invocation counts (own repo = NULL origin; foreign =
    # repo:/... from the cross-repo import).
    inv = q("""
      SELECT f.session_id AS sid, COALESCE(f.origin, '(own)') AS origin,
             COUNT(*) FILTER (WHERE t.cmd_prefix ILIKE 'rekal%') AS calls,
             COUNT(*) FILTER (WHERE t.cmd_prefix ILIKE 'rekal query%') AS drill
      FROM session_facets f
      LEFT JOIN tool_calls_index t ON t.session_id = f.session_id
      GROUP BY f.session_id, origin
    """)
    # Per-session human_steering counts.
    steer = {r["sid"]: int(r["n"]) for r in q(
        "SELECT session_id AS sid, COUNT(*) AS n FROM turns_ft "
        "WHERE role = 'human_steering' GROUP BY session_id"
    )}

    total = len(inv)
    using = [r for r in inv if int(r["calls"]) > 0]
    not_using = [r for r in inv if int(r["calls"]) == 0]
    n_using = len(using)

    calls_per_using = mean([int(r["calls"]) for r in using])
    drill_through = (
        sum(1 for r in using if int(r["drill"]) > 0) / n_using if n_using else 0.0
    )
    steer_using = mean([steer.get(r["sid"], 0) for r in using])
    steer_not = mean([steer.get(r["sid"], 0) for r in not_using])

    # Adoption by repo scope.
    repos_using = sorted({r["origin"] for r in using})

    agg = {
        "sessions_total": total,
        "sessions_using_rekal": n_using,
        "adoption_rate": round(n_using / total, 3) if total else 0.0,
        "repos_with_usage": len(repos_using),
        "rekal_calls_per_using_session": round(calls_per_using, 2),
        "drill_through_rate": round(drill_through, 3),
        "mean_steering_using": round(steer_using, 2),
        "mean_steering_not_using": round(steer_not, 2),
        "steering_delta": round(steer_using - steer_not, 2),
    }
    (outdir / "usage.json").write_text(json.dumps(agg, indent=2))

    lines = [
        "# Rekal usage & effectiveness (observational)",
        "",
        f"- **Adoption:** {n_using}/{total} sessions invoked `rekal` "
        f"({agg['adoption_rate']:.1%}), across {len(repos_using)} repo scope(s).",
        f"- **Intensity:** {calls_per_using:.2f} `rekal` calls per using-session.",
        f"- **Drill-through:** {drill_through:.1%} of using-sessions followed a "
        "search with `rekal query --session` (implicit relevance signal).",
        f"- **Steering delta (natural experiment):** {steer_using:.2f} "
        f"human_steering turns per rekal-using session vs {steer_not:.2f} "
        f"without (Δ {agg['steering_delta']:+.2f}).",
        "",
        "_Observational, not controlled: agents and task shapes differ between "
        "the two groups. Read as a large-N prior for the A/B tests (§4b), not "
        "as a causal effect._",
    ]
    (outdir / "usage.md").write_text("\n".join(lines) + "\n")
    print("\n".join(lines))


if __name__ == "__main__":
    main()
