# Architecture Decision Records

> **WHAT**: Append-only log of architectural decisions. One file per decision.
> **AUTHORITY**: 🏛️ HISTORICAL — past decisions are immutable. Superseding a decision means writing a new ADR that references the old one.

## Index

(No ADRs filed yet. The first ADR — `0001-tmdb-and-wikidata-as-canonical-sources.md` — should be retroactively written before the next data-source change.)

## When to write an ADR

Open a new file `NNNN-<slug>.md` (next-free numeric prefix) when:

- A CinovaScore parameter changes (Bayesian prior `m`, corpus mean `C`, blend ratio).
- A new data source is added (today: TMDB, Wikidata, YouTube).
- A new persistence layer is added or an existing one's role changes.
- A mobile architectural change affects multiple routes (state-management, routing-library swap).
- A security model changes (auth flow, secret store, identity provider).
- A deployment target changes (e.g., migrating from `contabo-mkt` Sovereign to a Hetzner Sovereign Blueprint).

## ADR template

```markdown
# NNNN — <Title>

| Status | Accepted / Superseded by NNNN |
| Date | YYYY-MM-DD |
| Author | <handle> |

## Context

<What was the situation that prompted this decision?>

## Decision

<What did we decide, in one paragraph?>

## Consequences

<What does this enable? What does it foreclose? What new risks does it create?>

## Alternatives considered

<List the realistic alternatives and why each was rejected.>

## Related

<Links to issues, PRs, prior ADRs.>
```

Once an ADR is merged with `Status: Accepted`, do not edit it. Superseding requires a new ADR that references the old one and the old one's status flips to `Superseded by NNNN`.
