# Event-time workflow

Use this lens when the answer requested is a date, event time, or temporal
relation. The target is event time, which may differ from message time.

**Evidence contract**

- The exact actor, event, endpoint, and requested precision stay fixed.
- A relative phrase in the question is anchored to the asker's present. A
  relative phrase in historical evidence is anchored to that assertion turn's
  timestamp. The record's newest turn is not automatically either anchor.
- Event time and assertion time stay separate. A resolvable source-relative
  phrase such as "yesterday" is converted before answering an event-time ask.
- Source precision is preserved: month-only evidence supports a month, not an
  invented day; approximate evidence supports an approximate answer.
- Every endpoint of a range or before/after relation needs independent support.

Useful operations include drilling the assertion turn, recording its timestamp
separately from its content, and using SQL or DuckDB date arithmetic for
weekday, month-boundary, leap-year, or range calculations.

If no explicit anchor supports conversion, keep the genuine ambiguity instead of
manufacturing precision. Otherwise answer with the resolved event date, range,
or calendar relation requested.
