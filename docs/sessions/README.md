# Session Artifacts

> **WHAT**: Date-stamped per-session artifacts — walk reports, retrospectives, audit reports, ad-hoc analysis. Named `<YYYY-MM-DD>-<topic>.md`.
> **AUTHORITY**: 🗓️ TRANSIENT. Kept for 30 days after the session ends. After 30 days, sessions with resolved TRUST.md entries move to [../archive/](../archive/); unresolved sessions remain with a note on [../STATUS.md](../STATUS.md). Sessions older than 90 days move to `archive/` unconditionally.

## Index

(Empty.) A session artifact is created when the session has produced something durable — a walk report with screenshots, a retrospective worth remembering, an audit of a specific surface.

## When NOT to create a session artifact

- Routine code work — the PR is the record.
- One-off questions — the issue thread is the record.
- Status updates — [../STATUS.md](../STATUS.md) and [../ledger/TRACKER.md](../ledger/TRACKER.md) are the record.

## Template

```markdown
# <YYYY-MM-DD> — <Topic>

| Session ID | <UUID> |
| Author | <handle> |
| Related issues | #N, #M |

## What I did

## What I found

## What changed

## What's left
```
