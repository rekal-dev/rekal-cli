# Duration workflow

Use this lens only when the answer requested is elapsed time or a duration
between endpoints. The arithmetic is the last step, not the retrieval method.

**Evidence contract**

- The exact start event and end event must each be supported independently. A
  nearby event involving the same topic is not an endpoint.
- Event time and mention time are separate. A source-relative phrase such as
  "three days ago" is anchored to the historical assertion turn.
- Both endpoints need compatible precision. Month-only or approximate evidence
  does not support a manufactured exact day.
- Direction, units, and inclusive/exclusive interpretation remain part of the
  requested relation, not details to infer from whichever hit ranked first.

Useful operations include separate recall and drilling for each endpoint and
DuckDB date arithmetic for calendar boundaries, weekdays, or intervals that are
easy to misread. The agent chooses the operations needed by the evidence.

If an endpoint remains unsupported after a focused reformulation, preserve the
gap rather than borrowing another event's date. Answer only the requested
duration.
