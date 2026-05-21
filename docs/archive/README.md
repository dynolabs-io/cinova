# Archive

> **WHAT**: Frozen / superseded / historical documents. Reference only — never auto-loaded by agents.
> **AUTHORITY**: 🗄️ FROZEN. Do not treat archive content as current. The pointer to the replacement (canonical doc or new ADR) lives at the top of each archived file.

## Index

(Empty.) Files arrive here when:

- A 📐 PERMANENT keeper is consolidated into another (the loser moves here).
- A 💡 proposal is rejected (the proposal file moves here).
- A 🗓️ session artifact ages past 90 days.
- A 🏛️ ADR is superseded — the original stays in `adr/` with `Superseded by NNNN`; only docs that didn't already have an immutable home move here.

## File-naming convention

```
<YYYY-MM-DD>-<topic-or-original-slug>.md
```

The leading date is the date the file was archived, not when it was originally authored. Each archived file MUST open with:

```markdown
> **ARCHIVED**: YYYY-MM-DD. This document is no longer authoritative.
> **Superseded by**: <relative path to current doc>
> **Reason**: <one-line why>
```
