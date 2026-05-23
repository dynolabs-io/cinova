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
| iOS app boots without native crash | 🟡 VERIFIED-PARTIAL | `2038bd3` build 14 | claude-cinova-session-222aa08c | 2026-05-21 | [docs/walks/2026-05-21-cinova-build14-launch.png](../walks/2026-05-21-cinova-build14-launch.png) — iOS Sim screenshot from ios.yml run 26224865258; same Cinova.app bytes as TestFlight build 14 |
| iOS tab bar renders (5 tabs + chat FAB) | 🟡 VERIFIED-PARTIAL | `2038bd3` build 14 | claude-cinova-session-222aa08c | 2026-05-21 | Same screenshot — 5 tab icons + chat FAB visible. **Tabs are icon-only, no text labels** (intentional design) |
| iOS app installable from TestFlight | 🟡 VERIFIED-PARTIAL | build 14 | claude | 2026-05-22 | Apple ASC: build 14 processingState=VALID, internalBuildState=IN_BETA_TESTING, attached to Founders group + both individualTesters (#106). iPhone-device install confirmation pending. |
| iOS Home tab — visual frame renders | 🟢 VERIFIED-PASS | `ff6730e` build 18 | claude-cinova-session-222aa08c | 2026-05-23 | [docs/walks/walk-1-home.png](../walks/walk-1-home.png) from ios.yml run 26331965196 |
| iOS Reels tab — visual frame renders | 🟢 VERIFIED-PASS | `ff6730e` build 18 | claude-cinova-session-222aa08c | 2026-05-23 | [docs/walks/walk-2-reels.png](../walks/walk-2-reels.png) |
| iOS Discover tab — visual frame renders | 🟢 VERIFIED-PASS | `ff6730e` build 18 | claude-cinova-session-222aa08c | 2026-05-23 | [docs/walks/walk-3-discover.png](../walks/walk-3-discover.png) |
| iOS Watchlist tab — visual frame renders | 🟢 VERIFIED-PASS | `ff6730e` build 18 | claude-cinova-session-222aa08c | 2026-05-23 | [docs/walks/walk-4-watchlist.png](../walks/walk-4-watchlist.png) |
| iOS Profile tab — visual frame renders | 🟢 VERIFIED-PASS | `ff6730e` build 18 | claude-cinova-session-222aa08c | 2026-05-23 | [docs/walks/walk-5-profile.png](../walks/walk-5-profile.png) |
| Home tab — trending feed renders within 2s | 🔴 UNVERIFIED | — | — | — | Backend `api.cinova.openova.io` scaled-to-0 (openova-private `faeffea2`); content area is blank by design until backend re-enabled |
| Home tab — tap poster → movie detail loads w/ CinovaScore | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| Discover (reels) — swipe through ≥ 5 vertical trailers autoplay | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| Discover (reels) — Save persists across app restart | 🔴 UNVERIFIED | — | — | — | — |
| Discover (reels) — country filter NOT applied unless user set | 🔴 UNVERIFIED | — | — | — | Last regression: `5f240fd` removed accidental filter |
| Search — NL query returns ≥ 3 hits | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| Search — tap a hit lands on detail | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| Watchlist — save from 2 surfaces, both visible after restart | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| AI chat — response cites ≥ 1 specific title with reasoning | 🔴 UNVERIFIED | — | — | — | Known bug: country hardcoded US (#84) — blocked on backend |
| Auth — anonymous → signup migrates watchlist | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| Auth — restart app, still logged in | 🔴 UNVERIFIED | — | — | — | Blocked on backend |
| API `/healthz` returns 200 | 🔴 UNVERIFIED | — | — | — | Backend scaled to 0 |
| API `/readyz` returns 200 (DB + cache check) | 🔴 UNVERIFIED | — | — | — | Backend scaled to 0 |

## Update protocol

When you walk a surface:

1. Edit the row above with the new state, SHA, your handle, the date, and the screenshot URL.
2. Comment on the related issue with the same evidence.
3. If state is 🟢, also update [TRACKER.md](TRACKER.md) to move the issue toward "awaiting user closure".
4. If state is ⛔ or 🟡, do not close anything — leave the open issue for a fix-author.

When a PR lands on a surface, set its state back to 🔴 UNVERIFIED in the same PR.
