# Complete-set and count workflow

Use this lens when the answer requested is a count, explicit or implicit plural
set, repeated events, or an ordered history. Ranked recall finds loud matches;
it does not prove completeness.

**Evidence contract**

- Actor, entity, class, time window, and set boundary define membership.
- Completeness requires a chronological sweep, not confidence in the top hits.
- Actual events or possessions remain distinct from plans, suggestions,
  comparisons, negations, and another person's facts.
- Duplicate mentions of one event are removed without collapsing distinct
  repeated events. A stated expected count does not force membership.

Useful operations include `rekal find` on a stable entity and SQL for
chronology, counting, or analytical sets; read both speakers and relevant
captions, links, quoted lists, and uptake turns. Page until empty when claiming
completeness.

Return only the requested count, items, or order; unsupported members do not
belong in the set.
