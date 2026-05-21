# Engineering Principles

> **WHAT**: Engineering rules that apply specifically to Cinova, in addition to the user-global rules at `~/.claude/CLAUDE.md`.
> **AUTHORITY**: 📐 Canon. When in doubt, user-global wins; this doc captures cinova-specific deltas + the anti-pattern catalog tied to actual PR receipts.
> **POINTER**: DoD criteria in [DOD.md](DOD.md). Trust ledger in [ledger/TRUST.md](ledger/TRUST.md).

## Inviolable, cinova-specific

These add to (never override) the user-global rules.

1. **Reels feed is fragile — revert before "fixing".** The vertical-trailer reels surface (mobile `(tabs)/discover`) has a documented regression history of well-intentioned "improvements" that broke autoplay or filtered out content. Default to reverting to the last green commit (e.g., the v12 baseline preserved in commits `7d865f9` / `180f453`) before attempting a forward fix. The fix-train of 2026-05 (commits `ca307a1` → `7d865f9` → `5f240fd`) is a case study in this pattern.
2. **No country filter on the reels feed** unless a user explicitly sets a country preference. Commit `5f240fd` removed an accidental filter that hid 21 valid vertical trailers; do not reintroduce.
3. **CinovaScore formula constants are not tunables-by-PR.** The Bayesian prior `m=1000`, corpus mean `C≈6.5`, and `(0.8, 0.2)` blend live in `backend/internal/scoring`. Any change requires an ADR in [adr/](adr/) and a separate DoD walk because the change touches every ranking surface.
4. **Production manifests live in `openova-private`, not this repo.** Do not add Helm charts, raw `kubectl apply` manifests, or Flux Kustomizations to this repo's `deploy/` directory until the Sovereign Blueprint migration ships. Drift between `deploy/` and `openova-private/clusters/contabo-mkt/apps/cinova/` is a debugging-hours sink.
5. **Mobile fonts must be preloaded before `useFonts()`-gated render.** Tab icons rendering as `?` glyphs (#77) was a font-preload regression. When adding a new icon font, register it in the mobile font-loader bootstrap before any consumer renders.
6. **Axon is the only path to Claude.** Backend Go code must never call `api.anthropic.com` directly — always go through the internal Axon gateway. This keeps cost + tracing centralised in Langfuse.
7. **JWT secrets are not committable.** `JWT_SECRET`, `TMDB_API_KEY`, database URLs, and the Axon token are sealed-secret-only. See [SECURITY.md](SECURITY.md).

## Anti-pattern catalog (with PR receipts)

Each row is a real pattern that has bitten this repo. The "wrong response" column is what *not* to do; the "right response" is what to do instead.

| Pattern | Wrong response | Right response | Receipts |
|---|---|---|---|
| Reels surface broken after a "small" tweak | Add a null guard + ship | Revert to last-known-green commit; investigate forward fix on a branch | Commits `7d865f9`, `180f453` (revert-to-v12) |
| Country filter silently scoping a feed | Add a default country fallback | Remove the filter; let server-side ranking handle preference | Commit `5f240fd` |
| Tab icons render `?` glyphs | Replace icon strings | Fix the font preload order | Issue #77, fix commit `e4bc827` |
| Movie detail "stuck on spinner" | Add a 5s timeout + show "Try again" | Trace the actual data-fetch path; usually a Neo4j query timing out | Issue #78 |
| Provider deep link opens App Store | Add a try/catch + JustWatch fallback | Fix the URL scheme detection upstream | Issue #91 |
| AI chat answers in English when user country is non-US | Hardcode the country in the chat handler | Plumb the user's country preference end-to-end (handler → Axon prompt) | Issue #84 |
| New "improvement" to a fragile surface | Ship it because tests pass | Walk DoD gate 4 before merging | Reels regression cluster (2026-05) |

## Defensive-coding triggers (from user-global §3 rule 3 — same as platform)

Each of these is a smell, not a fix. When you see one, investigate the upstream cause:

- Null-guards on empty data
- `?? 'default'` fallbacks
- `enabled: false` defaults
- `cursor: 'default'` style guards
- Snapshot-empty-frame test scaffolding
- `must_contain`-token-passing test churn

## Definition of "not a workaround"

Per user-global §4 rule 14, every fix must be the architecturally-correct end state. Cinova-specific examples:

| Workaround | Target-state fix |
|---|---|
| `if (videos.length === 0) return null` on reels | Trace why ingestion returned 0; fix the filter or the data |
| Pinning ingestion to a single country to dodge a Wikidata-rate-limit | Add a respectful retry + backoff in the SPARQL client |
| Hardcoding `country=US` in chat handler | Plumb user preference all the way through |
| Disabling an Expo plugin because EAS build failed | Fix the plugin config or the Xcode/Gradle entry it generates |

## When to write an ADR

Open a new file in [adr/](adr/) (next-free numeric prefix, `NNNN-<slug>.md`) when:

- A scoring or ranking parameter changes
- A new data source is added (today: TMDB, Wikidata, YouTube)
- A new persistence layer is added or an existing one's role changes
- A mobile architectural change affects multiple routes (e.g., switching state management library)
- A security model changes (auth, secrets, identity)

ADRs are append-only and immutable once accepted. Superseding an ADR means writing a new one that references the old one.
