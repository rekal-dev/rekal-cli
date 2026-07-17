#!/usr/bin/env python3
"""WS-A: normalize LoCoMo (locomo10.json) to the interchange format
(SCHEMA.md). Stdlib only, deterministic.

Mapping decisions (recorded here because they are judgement calls):

- LoCoMo is a dialogue between two named personas, not user/assistant. The
  first speaker of session_1 maps to role "user", the other to "assistant",
  and EVERY turn's text is prefixed "<Speaker>: " so recall can resolve the
  names the questions use. Both names are kept in extra.speakers.
- Image-sharing turns keep their text and append "[shares a photo: <blip
  caption>]" — captions carry answer-bearing content in LoCoMo.
- evidence appears both as a real list and as a string repr of a list;
  both are parsed. Dialogue ids "D<k>:<n>" map to session ids "s<k>"
  (session-level evidence; turn-level offsets go to extra.evidence_dia_ids).
- Categories (LoCoMo paper taxonomy): 1 multi-hop, 2 temporal,
  3 open-domain, 4 single-hop, 5 adversarial. Category 5's correct
  behavior is abstention: answer becomes "unknown",
  evidence_session_ids [], and the trap answer moves to
  extra.adversarial_answer.
- Session date strings like "1:56 pm on 8 May, 2023" parse to RFC3339 UTC.
"""

import argparse
import ast
import json
import re
import sys
from datetime import datetime, timezone

CATEGORY_NAMES = {
    "1": "multi-hop",
    "2": "temporal",
    "3": "open-domain",
    "4": "single-hop",
    "5": "adversarial",
}

DATE_RE = re.compile(r"(\d{1,2}):(\d{2})\s*(am|pm)\s+on\s+(\d{1,2})\s+(\w+),?\s+(\d{4})", re.I)


def parse_date(s: str) -> str:
    m = DATE_RE.search(s)
    if not m:
        raise ValueError(f"unparseable LoCoMo date: {s!r}")
    hh, mm, ap, day, month, year = m.groups()
    hour = int(hh) % 12 + (12 if ap.lower() == "pm" else 0)
    dt = datetime.strptime(f"{day} {month} {year}", "%d %B %Y").replace(
        hour=hour, minute=int(mm), tzinfo=timezone.utc
    )
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def parse_evidence(ev):
    if ev is None:
        return []
    if isinstance(ev, str):
        try:
            ev = ast.literal_eval(ev)
        except (ValueError, SyntaxError):
            return []
    if not isinstance(ev, list):
        return []
    return [e for e in ev if isinstance(e, str)]


def dia_to_session(dia_id: str):
    m = re.match(r"D(\d+):", dia_id)
    return f"s{m.group(1)}" if m else None


def turn_text(turn: dict) -> str:
    text = (turn.get("text") or "").strip()
    cap = (turn.get("blip_caption") or "").strip()
    if cap:
        text = f"{text} [shares a photo: {cap}]".strip()
    return f"{turn['speaker']}: {text}"


def normalize(data):
    out = []
    for c in data:
        conv = c["conversation"]
        nums = sorted(
            int(k.split("_")[1])
            for k, v in conv.items()
            if re.fullmatch(r"session_\d+", k) and isinstance(v, list)
        )
        first_speaker = conv[f"session_{nums[0]}"][0]["speaker"]
        speakers = {first_speaker: "user"}
        sessions = []
        for n in nums:
            turns = []
            for t in conv[f"session_{n}"]:
                sp = t["speaker"]
                if sp not in speakers:
                    speakers[sp] = "assistant"
                turns.append({"role": speakers[sp], "text": turn_text(t)})
            sessions.append(
                {
                    "session_id": f"s{n}",
                    "date": parse_date(conv[f"session_{n}_date_time"]),
                    "turns": turns,
                }
            )
        sessions.sort(key=lambda s: s["date"])

        questions = []
        for i, q in enumerate(c["qa"]):
            cat_raw = str(q.get("category", ""))
            cat = CATEGORY_NAMES.get(cat_raw, f"unknown-{cat_raw}")
            dia_ids = parse_evidence(q.get("evidence"))
            ev_sessions = sorted({s for s in (dia_to_session(d) for d in dia_ids) if s},
                                 key=lambda x: int(x[1:]))
            extra = {"category_raw": cat_raw, "evidence_dia_ids": dia_ids}
            if cat == "adversarial":
                answer = "unknown"
                ev_sessions = []
                if "adversarial_answer" in q:
                    extra["adversarial_answer"] = q["adversarial_answer"]
            else:
                answer = str(q.get("answer", ""))
            questions.append(
                {
                    "question_id": f"q{i+1}",
                    "category": cat,
                    "question": q["question"],
                    "answer": answer,
                    "evidence_session_ids": ev_sessions,
                    "extra": extra,
                }
            )

        out.append(
            {
                "conversation_id": c["sample_id"],
                "sessions": sessions,
                "questions": questions,
                "extra": {"speakers": speakers},
            }
        )
    return out


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--input", default=None, help="locomo10.json (default: data/locomo10.json next to this script)")
    ap.add_argument("--output", default=None, help="conversations.jsonl (default: data/locomo-conversations.jsonl)")
    args = ap.parse_args()

    import pathlib

    here = pathlib.Path(__file__).parent
    inp = pathlib.Path(args.input) if args.input else here / "data" / "locomo10.json"
    outp = pathlib.Path(args.output) if args.output else here / "data" / "locomo-conversations.jsonl"

    data = json.load(open(inp))
    convs = normalize(data)

    n_sess = sum(len(c["sessions"]) for c in convs)
    n_turn = sum(len(s["turns"]) for c in convs for s in c["sessions"])
    n_q = sum(len(c["questions"]) for c in convs)
    with open(outp, "w") as f:
        for c in convs:
            f.write(json.dumps(c, ensure_ascii=False) + "\n")
    print(f"wrote {outp}: conversations={len(convs)} sessions={n_sess} turns={n_turn} questions={n_q}")

    # Source-of-truth cross-check: counts must match the raw file exactly.
    raw_sess = sum(1 for c in data for k, v in c["conversation"].items()
                   if re.fullmatch(r"session_\d+", k) and isinstance(v, list))
    raw_turn = sum(len(v) for c in data for k, v in c["conversation"].items()
                   if re.fullmatch(r"session_\d+", k) and isinstance(v, list))
    raw_q = sum(len(c["qa"]) for c in data)
    ok = (n_sess, n_turn, n_q) == (raw_sess, raw_turn, raw_q)
    print(f"cross-check vs raw: sessions {raw_sess} turns {raw_turn} questions {raw_q} -> {'ok' if ok else 'MISMATCH'}")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
