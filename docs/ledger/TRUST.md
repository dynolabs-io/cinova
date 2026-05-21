# Trust Ledger — verification state per surface

> **WHAT**: Per-surface verification state. Every entry is 🔴 UNVERIFIED, 🟢 VERIFIED-PASS, ⛔ VERIFIED-FAIL, or 🟡 VERIFIED-PARTIAL.
> **AUTHORITY**: 🟢 LIVE STATE. Updated on every operator-walk and every PR that touches a surface.
> **POINTER**: Walk criteria per surface live in [DOD.md](DOD.md). Issue-by-issue progress lives in [TRACKER.md](TRACKER.md).

## How to read this ledger

- **🔴 UNVERIFIED** — never walked, OR walked but a subsequent PR landed on this surface. Default state.
- **🟢 VERIFIED-PASS** — walk attached to the issue, screenshot URL recorded, SHA captured. The walked SHA is the as-of state.
- **⛔ VERIFIED-FAIL** — walk attempted, surface broken. Issue stays open.
- **🟡 VERIFIED-PARTIAL** — walk attempted, some criteria pass and others fail. Issue stays open with notes.

Per user-global `~/.claude/CLAUDE.md` §3 rule 6: verification is READ-ONLY. The verifier does not ship fixes to make their own walk pass.

## Surfaces

| Surface | State | Walked SHA | Walked by | Date | Evidence |
|---|---|---|---|---|---|
| Home tab — trending feed renders within 2s | 🔴 UNVERIFIED | — | — | — | — |
| Home tab — tap poster → movie detail loads w/ CinovaScore | 🔴 UNVERIFIED | — | — | — | — |
| Discover (reels) — swipe through ≥ 5 vertical trailers autoplay | 🔴 UNVERIFIED | — | — | — | — |
| Discover (reels) — Save persists across app restart | 🔴 UNVERIFIED | — | — | — | — |
| Discover (reels) — country filter NOT applied unless user set | 🔴 UNVERIFIED | — | — | — | Last regression: `5f240fd` removed accidental filter |
| Search — NL query returns ≥ 3 hits | 🔴 UNVERIFIED | — | — | — | — |
| Search — tap a hit lands on detail | 🔴 UNVERIFIED | — | — | — | — |
| Watchlist — save from 2 surfaces, both visible after restart | 🔴 UNVERIFIED | — | — | — | — |
| AI chat — response cites ≥ 1 specific title with reasoning | 🔴 UNVERIFIED | — | — | — | Known bug: country hardcoded US (#84) |
| Auth — anonymous → signup migrates watchlist | 🔴 UNVERIFIED | — | — | — | — |
| Auth — restart app, still logged in | 🔴 UNVERIFIED | — | — | — | — |
| API `/healthz` returns 200 | 🔴 UNVERIFIED | — | — | — | — |
| API `/readyz` returns 200 (DB + cache check) | 🔴 UNVERIFIED | — | — | — | — |

## Update protocol

When you walk a surface:

1. Edit the row above with the new state, SHA, your handle, the date, and the screenshot URL.
2. Comment on the related issue with the same evidence.
3. If state is 🟢, also update [TRACKER.md](TRACKER.md) to move the issue toward "awaiting user closure".
4. If state is ⛔ or 🟡, do not close anything — leave the open issue for a fix-author.

When a PR lands on a surface, set its state back to 🔴 UNVERIFIED in the same PR.
