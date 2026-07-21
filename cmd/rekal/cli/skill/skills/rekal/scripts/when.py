#!/usr/bin/env python3
"""Resolve a relative time phrase to an absolute date — deterministic calendar.

`rekal-when <anchor-date> "<phrase>"` turns "last Saturday", "yesterday",
"three weeks ago", "a few days ago" into the absolute date — or an honest
window when the phrase is vague — computed from the anchor, the date the
phrase was *said*. Event time ≠ mention time: read the mention's `ts`, pass
its date as the anchor, and this returns when the event actually was, without
error-prone mental date math.

  rekal-when 2023-05-25 "last Saturday"    -> 2023-05-20 (Saturday)
  rekal-when 2023-05-25 "yesterday"         -> 2023-05-24 (Wednesday)
  rekal-when 2023-05-25 "a few days ago"    -> 2023-05-22..2023-05-24 (approx)

Pure function: no store, no network, no tuned constant — only the calendar.
The agent picks the anchor and judges the result; the script does the
arithmetic. Vague phrases return a labelled window rather than fake precision
(SOUL.md: keep the speaker's precision — don't round, don't over-specify).

Exit: 0 resolved, 1 phrase not recognized (say so; don't guess), 2 usage/bad date.
"""
from __future__ import annotations

import calendar
import re
import sys
from datetime import date, timedelta

WEEKDAYS = {
    "monday": 0, "tuesday": 1, "wednesday": 2, "thursday": 3,
    "friday": 4, "saturday": 5, "sunday": 6,
    "mon": 0, "tue": 1, "tues": 1, "wed": 2, "thu": 3, "thur": 3,
    "thurs": 3, "fri": 4, "sat": 5, "sun": 6,
}
NUMWORDS = {
    "a": 1, "an": 1, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
    "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10, "couple": 2,
}


def fmt(d: date) -> str:
    return f"{d.isoformat()} ({calendar.day_name[d.weekday()]})"


def window(a: date, b: date) -> str:
    lo, hi = sorted((a, b))
    return f"{lo.isoformat()}..{hi.isoformat()} (approx)"


def add_months(d: date, n: int) -> date:
    """d shifted by n calendar months (n may be negative); day clamped."""
    m0 = d.year * 12 + (d.month - 1) + n
    year, month = divmod(m0, 12)
    month += 1
    last = calendar.monthrange(year, month)[1]
    return date(year, month, min(d.day, last))


def month_bounds(d: date) -> tuple[date, date]:
    last = calendar.monthrange(d.year, d.month)[1]
    return date(d.year, d.month, 1), date(d.year, d.month, last)


def week_bounds(d: date) -> tuple[date, date]:
    """Monday..Sunday of d's ISO week."""
    mon = d - timedelta(days=d.weekday())
    return mon, mon + timedelta(days=6)


def last_weekday(a: date, target: int) -> date:
    delta = (a.weekday() - target) % 7 or 7  # strictly before the anchor
    return a - timedelta(days=delta)


def next_weekday(a: date, target: int) -> date:
    delta = (target - a.weekday()) % 7 or 7  # strictly after the anchor
    return a + timedelta(days=delta)


def nearest_past_weekday(a: date, target: int) -> date:
    return a - timedelta(days=(a.weekday() - target) % 7)


def count(word: str) -> int | None:
    if word.isdigit():
        return int(word)
    return NUMWORDS.get(word)


def resolve(a: date, phrase: str) -> str | None:
    p = " ".join(phrase.lower().strip().strip(".").split())

    fixed = {
        "today": a, "tonight": a,
        "yesterday": a - timedelta(days=1),
        "tomorrow": a + timedelta(days=1),
        "day before yesterday": a - timedelta(days=2),
        "the day before yesterday": a - timedelta(days=2),
        "day after tomorrow": a + timedelta(days=2),
        "the day after tomorrow": a + timedelta(days=2),
    }
    if p in fixed:
        return fmt(fixed[p])

    # Fuzzy windows — honest ranges, never fake a single day.
    fuzzy = {
        r"^(a few|several) days? ago$": (a - timedelta(days=3), a - timedelta(days=1)),
        r"^(a )?couple( of)? days? ago$": (a - timedelta(days=3), a - timedelta(days=1)),
        r"^(recently|the other day|a while ago)$": (a - timedelta(days=7), a - timedelta(days=1)),
        r"^(a few|several) weeks? ago$": (a - timedelta(days=28), a - timedelta(days=14)),
    }
    for pat, (lo, hi) in fuzzy.items():
        if re.match(pat, p):
            return window(lo, hi)

    m = re.match(r"^(last|this|next|past) (\w+)$", p)
    if m:
        rel, word = m.group(1), m.group(2)
        if word in WEEKDAYS:
            wd = WEEKDAYS[word]
            if rel in ("last", "past"):
                return fmt(last_weekday(a, wd))
            if rel == "next":
                return fmt(next_weekday(a, wd))
            return fmt(nearest_past_weekday(a, wd))  # "this <weekday>"
        if word == "week":
            base = a + timedelta(days=7 * {"last": -1, "past": -1, "this": 0, "next": 1}[rel])
            lo, hi = week_bounds(base)
            return f"{lo.isoformat()}..{hi.isoformat()} (week)"
        if word == "month":
            lo, hi = month_bounds(add_months(a, {"last": -1, "past": -1, "this": 0, "next": 1}[rel]))
            return f"{lo.isoformat()}..{hi.isoformat()} (month)"
        if word == "year":
            y = a.year + {"last": -1, "past": -1, "this": 0, "next": 1}[rel]
            return f"{y}-01-01..{y}-12-31 (year)"

    # Bare weekday -> nearest past occurrence (ledger events are usually past).
    if p in WEEKDAYS:
        return fmt(nearest_past_weekday(a, WEEKDAYS[p]))

    m = re.match(r"^(\w+) (day|week|month|year)s? (ago|from now|later)$", p)
    if m:
        n = count(m.group(1))
        if n is None:
            return None
        sign = -1 if m.group(3) == "ago" else 1
        unit = m.group(2)
        if unit == "day":
            return fmt(a + timedelta(days=sign * n))
        if unit == "week":
            return fmt(a + timedelta(days=sign * 7 * n))
        if unit == "month":
            return fmt(add_months(a, sign * n))
        if unit == "year":
            return fmt(add_months(a, sign * 12 * n))

    return None


def main() -> int:
    args = [x for x in sys.argv[1:] if x not in ("-h", "--help")]
    if len(args) != len(sys.argv) - 1 or len(args) != 2:
        print('usage: when.py <YYYY-MM-DD> "<relative phrase>"', file=sys.stderr)
        return 2
    try:
        anchor = date.fromisoformat(args[0].strip())
    except ValueError:
        print(f"when: bad anchor date {args[0]!r} — want YYYY-MM-DD", file=sys.stderr)
        return 2

    out = resolve(anchor, args[1])
    if out is None:
        print(f'when: phrase not recognized: "{args[1]}" (resolve it by hand, or drill the turn)',
              file=sys.stderr)
        return 1
    print(out)
    return 0


if __name__ == "__main__":
    sys.exit(main())
