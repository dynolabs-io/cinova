# In-flight Proposals

> **WHAT**: Design proposals that are not yet decisions. Active write-ups for "should we do X?" — promoted to [../adr/](../adr/) when accepted, moved to [../archive/](../archive/) when rejected.
> **AUTHORITY**: 💡 Proposals. Not authoritative; ADRs are.

## Index

(Empty.) Start a proposal when you have a non-trivial architectural call that needs sign-off before code lands.

## When to write a proposal

- A change that would require an ADR if accepted.
- A change that touches multiple repos (cinova + openova-private).
- A change that affects the user-facing product (CinovaScore tweaks, new surface, removed surface).
- A change with non-obvious cost or vendor implications.

## Template

```markdown
# Proposal: <Title>

| Status | Draft / In Review / Accepted (→ ADR NNNN) / Rejected (→ archive/) |
| Author | <handle> |
| Opened | YYYY-MM-DD |

## Problem

<What's broken or limiting? Cite concrete evidence — issues, traces, user reports.>

## Proposal

<Concrete shape of the change.>

## Trade-offs

| Dimension | Win | Cost |
|---|---|---|
| <e.g., latency> | <e.g., -100ms p99> | <e.g., +20% infra cost> |

## Alternatives considered

## Rollout plan

<How do we ship this without breaking the canonical walk?>
```

When a proposal is **Accepted**, copy it to `adr/NNNN-<slug>.md` (next-free prefix) with `Status: Accepted`, and delete the file from `proposals/`.

When a proposal is **Rejected**, move it to `archive/<YYYY-MM-DD>-proposal-<slug>.md`.
