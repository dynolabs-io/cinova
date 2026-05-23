# Per-incident Runbooks

> **WHAT**: Specific incident playbooks — "if you see X, do Y". Distinct from [../RUNBOOKS.md](../RUNBOOKS.md), which is generic operator how-to (setup, deploy, ingestion ops).
> **AUTHORITY**: 🛠️ Operator playbooks. One file per incident class; named `<incident-class>.md`.

## Index

(Empty.) The first incident-runbook to write is the one you're paged for next.

## When to add a runbook

- A failure mode you've now seen ≥ 2 times.
- A known-tricky recovery path (e.g., "Neo4j refuses to start after OOMKill — wipe-and-restore-from-backup procedure").
- A coordinated procedure across multiple repos (e.g., "rotate JWT_SECRET — invalidates all clients; coordinated comms required").

## Template

```markdown
# Incident: <Class>

| Severity | sev-1 / sev-2 / sev-3 |
| Surface | <what's affected from the user's POV> |
| Mean time to recover | <observed in past incidents> |
| Last seen | YYYY-MM-DD (PR #N) |

## Symptoms

<What does the operator see? Log lines, dashboard signals, user reports.>

## Triage

1. <step>
2. <step>

## Recovery

1. <step>
2. <step>

## Root-cause fixes shipped

- [PR #N](https://github.com/dynolabs-io/cinova/pull/N) — <what it fixed>
```
